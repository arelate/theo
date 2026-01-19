package cli

import (
	"errors"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func InstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	os := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		os = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	var langCode string
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	var downloadTypes []vangogh_integration.DownloadType
	if q.Has(vangogh_integration.DownloadTypeProperty) {
		dts := strings.Split(q.Get(vangogh_integration.DownloadTypeProperty), ",")
		downloadTypes = vangogh_integration.ParseManyDownloadTypes(dts)
	}

	ii := &InstallInfo{
		OperatingSystem: os,
		LangCode:        langCode,
		DownloadTypes:   downloadTypes,
		KeepDownloads:   q.Has("keep-downloads"),
		NoSteamShortcut: q.Has("no-steam-shortcut"),
		UseSteamAssets:  q.Has("steam-assets"),
		reveal:          q.Has("reveal"),
		verbose:         q.Has("verbose"),
		force:           q.Has("force"),
	}

	if q.Has("env") {
		ii.Env = strings.Split(q.Get("env"), ",")
	}

	return Install(id, ii)
}

func Install(id string, ii *InstallInfo) error {

	ia := nod.Begin("installing %s...", id)
	defer ia.Done()

	if len(ii.DownloadTypes) == 1 && ii.DownloadTypes[0] == vangogh_integration.AnyDownloadType {
		ii.DownloadTypes = []vangogh_integration.DownloadType{vangogh_integration.Installer, vangogh_integration.DLC}
	}

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	printInstallInfoParams(ii, true, id)

	// always getting the latest product details for install purposes
	productDetails, err := getProductDetails(id, rdx, true)
	if err != nil {
		return err
	}

	switch productDetails.ProductType {
	case vangogh_integration.DlcProductType:
		ia.EndWithResult("install %s required product(s) to get this downloadable content", strings.Join(productDetails.RequiresGames, ","))
		return nil
	case vangogh_integration.PackProductType:
		ia.EndWithResult("installing product(s) included in this pack: %s", strings.Join(productDetails.IncludesGames, ","))
		for _, includedId := range productDetails.IncludesGames {
			if err = Install(includedId, ii); err != nil {
				return err
			}
		}
		return nil
	case vangogh_integration.GameProductType:
		// do nothing
	default:
		return errors.New("unknown product type " + productDetails.ProductType)
	}

	if err = resolveInstallInfo(id, ii, productDetails, rdx, currentOsThenWindows); err != nil {
		return err
	}

	ii.AddProductDetails(productDetails)

	// don't check existing installations for DLCs, Extras
	if slices.Contains(ii.DownloadTypes, vangogh_integration.Installer) && !ii.force {

		if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

			var installInfo *InstallInfo
			installInfo, _, err = matchInstallInfo(ii, installedInfoLines...)
			if err != nil {
				return err
			}

			if installInfo != nil {
				ia.EndWithResult("product %s is already installed", id)
				return nil
			}

		}

	}

	if err = BackupMetadata(); err != nil {
		return err
	}

	if err = Download(id, ii, nil, rdx); err != nil {
		return err
	}

	if err = Validate(id, ii, nil, rdx); err != nil {
		return err
	}

	if err = installProduct(id, ii, productDetails, rdx); err != nil {
		return err
	}

	if !ii.NoSteamShortcut {

		sgo := &steamGridOptions{
			useSteamAssets: ii.UseSteamAssets,
			logoPosition:   defaultLogoPosition(),
		}

		if err = addSteamShortcut(id, ii.OperatingSystem, ii.LangCode, sgo, rdx, ii.force); err != nil {
			return err
		}
	}

	if !ii.KeepDownloads {
		if err = RemoveDownloads(id, ii, rdx); err != nil {
			return err
		}
	}

	if err = pinInstallInfo(id, ii, rdx); err != nil {
		return err
	}

	idInstalledDate := map[string][]string{id: {time.Now().UTC().Format(time.RFC3339)}}
	if err = rdx.BatchReplaceValues(data.InstallDateProperty, idInstalledDate); err != nil {
		return err
	}

	if ii.reveal {
		if err = revealInstalled(id, ii); err != nil {
			return err
		}
	}

	return nil
}

func installProduct(id string, ii *InstallInfo, productDetails *vangogh_integration.ProductDetails, rdx redux.Writeable) error {

	ipa := nod.Begin("installing %s %s-%s...", id, ii.OperatingSystem, ii.LangCode)
	defer ipa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		ipa.EndWithResult("no links are matching install params")
		return nil
	}

	dlcNames := make(map[string]any)

	for _, dl := range dls {
		if ii.OperatingSystem != dl.OperatingSystem ||
			ii.LangCode != dl.LanguageCode {
			continue
		}
		if dl.DownloadType == vangogh_integration.DLC {
			dlcNames[dl.Name] = nil
		}
	}

	if len(dlcNames) > 0 {
		ii.DownloadableContent = slices.Collect(maps.Keys(dlcNames))
	}

	// installation:
	// 1. check available space
	// 2. perform pre-install actions (e.g. make setup executable on Linux)
	// 3. get protected locations files (e.g. Desktop shortcuts on Linux)
	// 4. unpack installers (e.g. pkgutil on macOS, execute .sh on Linux; run setup on Windows)
	// 5. perform post-unpack actions (e.g. reduce bundleName on macOS)
	// 5. uninstall if installed directory exists and forcing install (will be used for updates)
	// 6. create inventory of unpacked files
	// 7. place (move unpacked to install folder)
	// 8. perform post-install actions (e.g. run post-install script and remove xattrs on macOS)
	// 9. cleanup protected locations
	// 10. cleanup unpack directory

	// 1
	installedAppsDir := data.Pwd.AbsDirPath(data.InstalledApps)

	if err := hasFreeSpaceForProduct(productDetails, installedAppsDir, ii, nil); err != nil {
		return err
	}

	// 2
	if err := osPreInstallActions(id, ii, rdx); err != nil {
		return err
	}

	// 3
	preInstallFiles, err := osGetProtectedLocationsFiles(ii)
	if err != nil {
		return err
	}

	// 4
	unpackDir, err := osGetUnpackDir(id, ii, rdx)
	if err != nil {
		return err
	}

	if err = osUnpackInstallers(id, ii, dls, rdx, unpackDir); err != nil {
		return err
	}

	// 5
	if err = osPostUnpackActions(id, ii, dls, unpackDir, rdx); err != nil {
		return err
	}

	// 6
	absInstalledDir, err := osInstalledPath(id, ii.LangCode, ii.OperatingSystem, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absInstalledDir); err == nil && ii.force {
		if err = osUninstallProduct(id, ii, rdx); err != nil {
			return err
		}
	}

	// 7
	unpackedInventory, err := osGetInventory(id, ii, dls, rdx, unpackDir)
	if err != nil {
		return err
	}

	if err = writeInventory(id, ii.LangCode, ii.OperatingSystem, rdx, unpackedInventory...); err != nil {
		return err
	}

	// 8
	if err = osPlaceUnpackedFiles(id, ii, dls, rdx, unpackDir); err != nil {
		return err
	}

	// 9
	if err = osPostInstallActions(id, ii, dls, rdx, unpackDir); err != nil {
		return err
	}

	// 10
	postInstallFiles, err := osGetProtectedLocationsFiles(ii)
	if err != nil {
		return err
	}

	if err = removeNewFiles(preInstallFiles, postInstallFiles); err != nil {
		return err
	}

	// 11
	if err = os.RemoveAll(unpackDir); err != nil {
		return err
	}

	return nil
}

func osPreInstallActions(id string, ii *InstallInfo, rdx redux.Readable) error {

	switch ii.OperatingSystem {
	case vangogh_integration.Windows:
		switch data.CurrentOs() {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			return prefixInit(id, rdx, ii.verbose)
		default:
			return nil
		}
	default:
		return nil
	}
}

func osGetProtectedLocationsFiles(ii *InstallInfo) ([]string, error) {

	switch ii.OperatingSystem {
	case vangogh_integration.Linux:
		return linuxSnapshotDesktopFiles()
	default:
		return nil, nil
	}
}

func osGetUnpackDir(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {

	unpackDir := filepath.Join(data.Pwd.AbsDirPath(data.Temp), id)

	switch ii.OperatingSystem {
	case vangogh_integration.Windows:
		switch data.CurrentOs() {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			absPrefixDir, err := data.AbsPrefixDir(id, rdx)
			if err != nil {
				return "", err
			}
			return filepath.Join(absPrefixDir, prefixRelDriveCDir, "Temp", id), nil
		default:
			// do nothing
		}
	default:
		// do nothing
	}
	return unpackDir, nil
}

func osUnpackInstallers(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Writeable, unpackDir string) error {

	if _, err := os.Stat(unpackDir); err == nil {
		if ii.force {
			if err = os.RemoveAll(unpackDir); err != nil {
				return err
			}
		} else {
			return nil
		}
	}

	if _, err := os.Stat(unpackDir); os.IsNotExist(err) {
		if err = os.MkdirAll(unpackDir, 0755); err != nil {
			return err
		}
	}

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsUnpackInstallers(id, dls, unpackDir)
	case vangogh_integration.Linux:
		return linuxExecuteInstallers(id, dls, unpackDir)
	case vangogh_integration.Windows:
		switch data.CurrentOs() {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			return prefixUnpackInstallers(id, ii, dls, rdx, unpackDir)
		default:
			return ii.OperatingSystem.ErrUnsupported()
		}
	default:
		return ii.OperatingSystem.ErrUnsupported()
	}
}

func osPostUnpackActions(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, unpackDir string, rdx redux.Writeable) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsReduceBundleNameProperty(id, dls, unpackDir, rdx)
	case vangogh_integration.Linux:
		return linuxRemoveMojoSetupDirs(id, dls, unpackDir)
	default:
		return nil
	}
}

func osGetInventory(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Readable, unpackDir string) ([]string, error) {

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsGetInventory(id, dls, rdx, unpackDir)
	default:
		return getInventory(dls, unpackDir)
	}
}

func osPlaceUnpackedFiles(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Writeable, unpackDir string) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsPlaceUnpackedFiles(id, dls, rdx, unpackDir)
	case vangogh_integration.Linux:
		return linuxPlaceUnpackedFiles(id, dls, rdx, unpackDir)
	case vangogh_integration.Windows:
		switch data.CurrentOs() {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			return prefixPlaceUnpackedFiles(id, dls, rdx, unpackDir)
		default:
			return ii.OperatingSystem.ErrUnsupported()
		}
	default:
		return ii.OperatingSystem.ErrUnsupported()
	}
}

func placeUnpackedLinkPayload(link *vangogh_integration.ProductDownloadLink, absUnpackedPath, absInstallationPath string) error {

	mpda := nod.Begin(" placing unpacked %s files...", link.LocalFilename)
	defer mpda.Done()

	if _, err := os.Stat(absInstallationPath); os.IsNotExist(err) {
		if err = os.MkdirAll(absInstallationPath, 0755); err != nil {
			return err
		}
	}

	// enumerate all files in the payload directory
	relFiles, err := relWalkDir(absUnpackedPath)
	if err != nil {
		return err
	}

	for _, relFile := range relFiles {

		absSrcPath := filepath.Join(absUnpackedPath, relFile)

		absDstPath := filepath.Join(absInstallationPath, relFile)
		absDstDir, _ := filepath.Split(absDstPath)

		if _, err = os.Stat(absDstDir); os.IsNotExist(err) {
			if err = os.MkdirAll(absDstDir, 0755); err != nil {
				return err
			}
		}

		if err = os.Rename(absSrcPath, absDstPath); err != nil {
			return err
		}
	}

	return nil
}

func osPostInstallActions(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Readable, unpackDir string) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsPostInstallActions(id, dls, rdx, unpackDir)
	default:
		return nil
	}
}

func removeNewFiles(srcSet, newSet []string) error {

	for _, pidf := range newSet {

		if slices.Contains(srcSet, pidf) {
			continue
		}

		if err := os.Remove(pidf); err != nil {
			return err
		}
	}

	return nil
}

func osInstalledPath(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {

	installedAppsDir := data.Pwd.AbsDirPath(data.InstalledApps)

	osLangInstalledAppsDir := filepath.Join(installedAppsDir, data.OsLangCode(operatingSystem, langCode))

	if err := rdx.MustHave(vangogh_integration.SlugProperty, data.BundleNameProperty); err != nil {
		return "", err
	}

	var installedPath string
	if slug, ok := rdx.GetLastVal(vangogh_integration.SlugProperty, id); ok && slug != "" {
		installedPath = slug
	} else {
		return "", errors.New("slug is not defined for product " + id)
	}

	switch operatingSystem {
	case vangogh_integration.MacOS:
		if bundleName, sure := rdx.GetLastVal(data.BundleNameProperty, id); sure && bundleName != "" {
			installedPath = filepath.Join(installedPath, bundleName)
		}
	default:
		// do nothing
	}

	return filepath.Join(osLangInstalledAppsDir, installedPath), nil
}
