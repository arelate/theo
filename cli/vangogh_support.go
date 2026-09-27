package cli

import (
	"crypto/md5"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/author"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func vangoghDownloadsListSize(downloadsList vangogh_integration.DownloadsList, ii *InstallInfo, manualUrlFilter ...string) int64 {
	var totalEstimatedBytes int64

	downloadTypes := []vangogh_integration.DownloadType{vangogh_integration.Installer}
	if !ii.NoDlcs {
		downloadTypes = append(downloadTypes, vangogh_integration.DLC)
	}

	dls := downloadsList.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLangCodes(ii.LangCode).
		FilterDownloadTypes(downloadTypes...).
		FilterPatches(true)

	for _, dl := range dls {
		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}
		totalEstimatedBytes += dl.EstimatedBytes
	}

	return totalEstimatedBytes
}

func vangoghUninstallProduct(id string, ii *InstallInfo, rdx redux.Writeable) error {

	oupa := nod.Begin(" uninstalling %s %s-%s...", id, ii.OperatingSystem, ii.LangCode)
	defer oupa.Done()

	if err := removeInventoriedFiles(id, ii, rdx); err != nil {
		return err
	}

	return removeInventoryFile(id, ii, rdx)
}

func vangoghShortcutAssets(gogAssets map[string]string, rdx redux.Readable) (map[steam_grid.Asset]*url.URL, error) {

	shortcutAssets := make(map[steam_grid.Asset]*url.URL)

	for _, asset := range steam_grid.ShortcutAssets {

		var imageId string
		switch asset {
		case steam_grid.Header:
			imageId = gogAssets[vangogh_integration.GogHorizontalImageProperty]
		case steam_grid.LibraryCapsule:
			imageId = gogAssets[vangogh_integration.GogVerticalImageProperty]
		case steam_grid.LibraryHero:
			if heroImages := gogAssets[vangogh_integration.GogHeroProperty]; len(heroImages) > 0 {
				imageId = heroImages
			} else {
				imageId = gogAssets[vangogh_integration.GogBackgroundProperty]
			}
		case steam_grid.LibraryLogo:
			imageId = gogAssets[vangogh_integration.GogLogoProperty]
		case steam_grid.ClientIcon:
			if iconSquareImages := gogAssets[vangogh_integration.GogIconSquareProperty]; len(iconSquareImages) > 0 {
				imageId = iconSquareImages
			} else {
				imageId = gogAssets[vangogh_integration.GogIconProperty]
			}
		default:
			return nil, errors.New("unexpected shortcut asset " + asset.String())
		}

		if imageId != "" {

			apiImagePath := path.Join(data.ApiGogImagePath, imageId)
			vangoghImageUrl, err := data.VangoghUrl(apiImagePath, nil, rdx)
			if err != nil {
				return nil, err
			}

			shortcutAssets[asset] = vangoghImageUrl
		}
	}

	return shortcutAssets, nil

}

func vangoghSetupConnection(urlStr, username, password string, rdx redux.Writeable, reset bool) error {

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	if reset {
		if err := vangoghResetConnection(rdx); err != nil {
			return err
		}
	}

	if err := rdx.ReplaceValues(data.VangoghUrlProperty, data.VangoghUrlProperty, urlStr); err != nil {
		return err
	}

	if err := rdx.ReplaceValues(data.VangoghUsernameProperty, data.VangoghUsernameProperty, username); err != nil {
		return err
	}

	if err := vangoghUpdateSessionToken(password, rdx); err != nil {
		return err
	}

	return vangoghValidateSessionToken(rdx)
}

func vangoghResetConnection(rdx redux.Writeable) error {
	rvca := nod.Begin("resetting vangogh connection...")
	defer rvca.Done()

	for _, vp := range data.VangoghProperties() {
		if err := rdx.CutKeys(vp, vp); err != nil {
			return err
		}
	}

	return nil
}

func vangoghValidateSessionToken(rdx redux.Readable) error {

	tsa := nod.Begin("validating vangogh session token...")
	defer tsa.Done()

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	req, err := data.VangoghApiRequest(http.MethodPost, data.ApiAuthSessionPath, nil, rdx)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		msg := "session is not valid, please connect again"
		tsa.EndWithResult(msg)
		return errors.New(msg)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	var ste author.SessionTokenExpires

	if err = json.UnmarshalRead(resp.Body, &ste); err != nil {
		return err
	}

	utcNow := time.Now().UTC()

	if utcNow.Before(ste.Expires.Add(-1 * time.Hour * author.SessionNearExpirationHours)) {
		tsa.EndWithResult("session is valid")
		return nil
	} else {
		msg := "vangogh session expired or expires soon, connect to update"
		tsa.EndWithResult(msg)
		return errors.New(msg)
	}

}

func vangoghUpdateSessionToken(password string, rdx redux.Writeable) error {
	rsa := nod.Begin("updating vangogh session token...")
	defer rsa.Done()

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	var username string
	if up, ok := rdx.GetLastVal(data.VangoghUsernameProperty, data.VangoghUsernameProperty); ok && up != "" {
		username = up
	} else {
		return errors.New("username not found")
	}

	usernamePassword := url.Values{}
	usernamePassword.Set(vangogh_integration.UrlUsernameParameter, username)
	usernamePassword.Set(vangogh_integration.UrlPasswordParameter, password)

	req, err := data.VangoghApiRequest(http.MethodPost, data.ApiAuthUserPath, usernamePassword, rdx)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	var ste author.SessionTokenExpires

	if err = json.UnmarshalRead(resp.Body, &ste); err != nil {
		return err
	}

	if err = rdx.ReplaceValues(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty, ste.Token); err != nil {
		return err
	}

	if err = rdx.ReplaceValues(data.VangoghSessionExpiresProperty, data.VangoghSessionExpiresProperty, ste.Expires.Format(http.TimeFormat)); err != nil {
		return err
	}

	return nil
}

func vangoghUnpackPlace(id string, ii *InstallInfo, dt vangogh_integration.DownloadType, originData *data.OriginData, rdx redux.Writeable) error {

	ipa := nod.Begin("unpacking and placing %s %s-%s...", id, ii.OperatingSystem, ii.LangCode)
	defer ipa.Done()

	downloadsList, err := vangogh_integration.FromDetails(originData.GogDetails)
	if err != nil {
		return err
	}

	downloadsList = downloadsList.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLangCodes(ii.LangCode).
		FilterDownloadTypes(dt).
		FilterPatches(true)

	if dt == vangogh_integration.DLC {
		dlcNames := make(map[string]any)

		for _, dl := range downloadsList {
			if dl.DownloadType == vangogh_integration.DLC {
				dlcNames[dl.Name] = nil
			}
		}

		if len(dlcNames) > 0 {
			ii.DownloadableContent = slices.Collect(maps.Keys(dlcNames))
		}
	}

	localFilenames := gogDownloadslocalFilenames(downloadsList, originData.GogFilenames)

	if len(downloadsList) == 0 {
		ipa.EndWithResult("no links are matching install params")
		return nil
	}

	// vangogh installation:
	// 1. check available space
	// 2. unpack installers (e.g. pkgutil on macOS, extract .sh on Linux; innoextract/run setup on Windows)
	// 3. perform post-unpack actions (e.g. reduce bundleName on macOS)
	// 4. uninstall if installed directory exists and forcing install (will be used for updates)
	// 5. create inventory of unpacked files
	// 6. place (move unpacked to install folder)
	// 7. perform post-install actions (e.g. run post-install script and remove xattrs on macOS)
	// 8. cleanup unpack directory

	// 1
	installedAppsDir := camino.GetRel(vangogh_integration.GogApps, vangogh_integration.InstalledApps)

	if err = originHasFreeSpace(id, installedAppsDir, ii, originData); err != nil {
		return err
	}

	// 2
	unpackDir, err := vangoghGetUnpackDir(id, ii, rdx)
	if err != nil {
		return err
	}

	if err = vangoghUnpackInstallers(id, ii, downloadsList, originData.GogFilenames, rdx, unpackDir); err != nil {
		return err
	}

	// 3
	if err = vangoghPostUnpackActions(id, ii, localFilenames, unpackDir, rdx); err != nil {
		return err
	}

	// 4
	absInstalledDir, err := originOsInstalledPath(id, ii, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absInstalledDir); err == nil && ii.force {
		if err = vangoghUninstallProduct(id, ii, rdx); err != nil {
			return err
		}
	}

	// 5
	unpackedInventory, err := vangoghGetInventory(ii, downloadsList, originData.GogFilenames, unpackDir)
	if err != nil {
		return err
	}

	if err = appendInventory(id, ii.LangCode, ii.OperatingSystem, rdx, unpackedInventory...); err != nil {
		return err
	}

	// 6
	if err = vangoghPlaceUnpackedFiles(id, ii, downloadsList, originData.GogFilenames, rdx, unpackDir); err != nil {
		return err
	}

	// 7
	if err = vangoghPostInstallActions(id, ii, downloadsList, originData.GogFilenames, rdx, unpackDir); err != nil {
		return err
	}

	// 8
	if err = os.RemoveAll(unpackDir); err != nil {
		return err
	}

	return nil
}

func vangoghGetUnpackDir(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {

	unpackDir := filepath.Join(camino.GetRel(vangogh_integration.Temp, vangogh_integration.Downloads), id)

	switch ii.OperatingSystem {
	case vangogh_integration.Windows:
		switch vangogh_integration.CurrentOs() {
		case vangogh_integration.MacOS:

			ieInstalled := macOsIsInnoextractInstalled()
			switch ieInstalled {
			case true:
				// do nothing
			case false:
				return prefixTempUnpackDir(id, ii.Origin, rdx)
			}

		case vangogh_integration.Linux:
			return prefixTempUnpackDir(id, ii.Origin, rdx)
		default:
			// do nothing
		}
	default:
		// do nothing
	}
	return unpackDir, nil
}

func vangoghUnpackInstallers(id string, ii *InstallInfo, downloadsList vangogh_integration.DownloadsList, gogFilenames map[string]string, rdx redux.Writeable, unpackDir string) error {

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
		if err = os.MkdirAll(unpackDir, camino.DefaultFileMode); err != nil {
			return err
		}
	}

	localFilenames := gogDownloadslocalFilenames(downloadsList, gogFilenames)

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsUnpackInstallers(id, downloadsList, gogFilenames, unpackDir, ii.force)
	case vangogh_integration.Linux:
		return linuxUnpackInstallers(id, localFilenames, unpackDir)
	case vangogh_integration.Windows:
		switch vangogh_integration.CurrentOs() {
		case vangogh_integration.MacOS:
			return macOsUnpackWindowsInstallers(id, ii, localFilenames, rdx, unpackDir)
		case vangogh_integration.Linux:
			return linuxUnpackWindowsInstallers(id, ii, localFilenames, rdx, unpackDir)
		default:
			return ii.OperatingSystem.ErrUnsupported()
		}
	default:
		return ii.OperatingSystem.ErrUnsupported()
	}
}

func vangoghPostUnpackActions(id string, ii *InstallInfo, localFilenames []string, unpackDir string, rdx redux.Writeable) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsReduceBundleNameProperty(id, localFilenames, unpackDir, rdx)
	default:
		return nil
	}
}

func vangoghGetInventory(ii *InstallInfo, downloadsList vangogh_integration.DownloadsList, gogFilenames map[string]string, unpackDir string) ([]string, error) {

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return vangoghMacOsGetInventory(downloadsList, gogFilenames, unpackDir, ii.force)
	default:
		return vangoghGetOsInventory(ii.OperatingSystem, downloadsList, gogFilenames, unpackDir)
	}
}

func vangoghPlaceUnpackedFiles(id string, ii *InstallInfo, downloadsList vangogh_integration.DownloadsList, gogFilenames map[string]string, rdx redux.Writeable, unpackDir string) error {

	localFilenames := gogDownloadslocalFilenames(downloadsList, gogFilenames)

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsPlaceUnpackedFiles(id, ii, downloadsList, gogFilenames, rdx, unpackDir, ii.force)
	case vangogh_integration.Linux:
		return linuxPlaceUnpackedFiles(id, ii, localFilenames, rdx, unpackDir)
	case vangogh_integration.Windows:
		switch vangogh_integration.CurrentOs() {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			return prefixPlaceUnpackedFiles(id, ii, localFilenames, rdx, unpackDir)
		default:
			return ii.OperatingSystem.ErrUnsupported()
		}
	default:
		return ii.OperatingSystem.ErrUnsupported()
	}
}

func vangoghPlaceUnpackedLinkPayload(localFilename string, absUnpackedPath, absInstallationPath string) error {

	mpda := nod.Begin(" placing unpacked %s files...", localFilename)
	defer mpda.Done()

	if _, err := os.Stat(absInstallationPath); os.IsNotExist(err) {
		if err = os.MkdirAll(absInstallationPath, camino.DefaultFileMode); err != nil {
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
			if err = os.MkdirAll(absDstDir, camino.DefaultFileMode); err != nil {
				return err
			}
		}

		if err = os.Rename(absSrcPath, absDstPath); err != nil {
			return err
		}
	}

	return nil
}

func vangoghPostInstallActions(id string, ii *InstallInfo, downloadsList vangogh_integration.DownloadsList, gogFilenames map[string]string, rdx redux.Readable, unpackDir string) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsPostInstallActions(id, ii, downloadsList, gogFilenames, rdx, unpackDir, ii.force)
	default:
		return nil
	}
}

func vangoghDownloadData(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable, manualUrlFilter ...string) error {

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	downloadsDir := camino.GetAbs(vangogh_integration.Downloads)

	if err := originHasFreeSpace(id, downloadsDir, ii, originData, manualUrlFilter...); err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if token, ok := rdx.GetLastVal(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty); ok && token != "" {
		dc.SetAuthorizationBearer(token)
	}

	downloadTypes := []vangogh_integration.DownloadType{vangogh_integration.Installer}
	switch ii.NoDlcs {
	case false:
		downloadTypes = append(downloadTypes, vangogh_integration.DLC)
	default: // no nothing
	}

	downloadsList, err := vangogh_integration.FromDetails(originData.GogDetails)
	if err != nil {
		return err
	}

	downloadsList = downloadsList.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLangCodes(ii.LangCode).
		FilterDownloadTypes(downloadTypes...).
		FilterPatches(true)

	if len(downloadsList) == 0 {
		return errors.New("no links are matching operating params")
	}

	for _, dl := range downloadsList {

		var localFilename string
		if localFilename = originData.GogFilenames[dl.ManualUrl]; localFilename == "" {
			return errors.New("unresolved local filename for manual-url " + dl.ManualUrl)
		}

		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}

		fa := nod.NewProgress(" - %s...", localFilename)

		manualUrlPath := path.Join(data.ApiGogManualUrlPath, id, dl.DownloadType.String(), dl.ManualUrl)

		fileUrl, err := data.VangoghUrl(manualUrlPath, nil, rdx)
		if err != nil {
			fa.EndWithResult(err.Error())
			continue
		}

		if err = dc.Download(fileUrl, ii.force, fa, downloadsDir, id, localFilename); err != nil {
			fa.EndWithResult(err.Error())
			continue
		}

		fa.Done()
	}

	return nil
}

func vangoghRemoveProductDownloadLinks(id string,
	originData *data.OriginData,
	ii *InstallInfo,
	downloadsDir string) error {

	rdla := nod.Begin(" removing downloads for %s...", id)
	defer rdla.Done()

	idPath := filepath.Join(downloadsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rdla.EndWithResult("product downloads dir not present")
		return nil
	}

	downloadTypes := []vangogh_integration.DownloadType{vangogh_integration.Installer}
	if !ii.NoDlcs {
		downloadTypes = append(downloadTypes, vangogh_integration.DLC)
	}

	downloadsList, err := vangogh_integration.FromDetails(originData.GogDetails)
	if err != nil {
		return err
	}

	downloadsList = downloadsList.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLangCodes(ii.LangCode).
		FilterDownloadTypes(downloadTypes...)

	if len(downloadsList) == 0 {
		rdla.EndWithResult("no links are matching operating params")
		return nil
	}

	for _, dl := range downloadsList {

		var localFilename string
		// if we don't do this - product downloads dir itself will be removed
		if localFilename = originData.GogFilenames[dl.ManualUrl]; localFilename == "" {
			continue
		}

		absPath := filepath.Join(downloadsDir, id, localFilename)

		fa := nod.NewProgress(" - %s...", localFilename)

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fa.EndWithResult("not present")
			continue
		}

		if err := os.Remove(absPath); err != nil {
			return err
		}

		fa.Done()
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)
	if entries, err := os.ReadDir(productDownloadsDir); err == nil && len(entries) == 0 {
		rdda := nod.Begin(" removing empty product downloads directory...")
		if err = os.Remove(productDownloadsDir); err != nil {
			return err
		}
		rdda.Done()
	} else {
		return err
	}

	return nil
}

func vangoghGetExecTask(id string, ii *InstallInfo, rdx redux.Readable, et *execTask) (*execTask, error) {

	var err error
	if err = osConfirmRunnability(ii.OperatingSystem); err != nil {
		return nil, err
	}

	if ii.OperatingSystem == vangogh_integration.Windows && vangogh_integration.CurrentOs() != vangogh_integration.Windows {

		var absPrefixDir string
		if absPrefixDir, err = data.AbsPrefixDir(id, ii.Origin, rdx); err == nil {
			et.prefix = absPrefixDir
		} else {
			return nil, err
		}

		if et.exe != "" {
			return et, nil
		}
	}

	var absGogGameInfoPath string
	switch et.defaultLauncher {
	case false:
		absGogGameInfoPath, err = osFindGogGameInfo(id, ii, rdx)
		if err != nil {
			return nil, err
		}
	case true:
		// do nothing
	}

	switch absGogGameInfoPath {
	case "":
		var absDefaultLauncherPath string
		if absDefaultLauncherPath, err = osFindDefaultLauncher(id, ii, rdx); err != nil {
			return nil, err
		}
		if et, err = osExecTaskDefaultLauncher(absDefaultLauncherPath, ii.OperatingSystem, et); err != nil {
			return nil, err
		}
	default:
		if et, err = osExecTaskGogGameInfo(absGogGameInfoPath, ii.OperatingSystem, et); err != nil {
			return nil, err
		}
	}

	return et, nil
}

func vangoghValidateData(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable, manualUrlFilter ...string) error {
	va := nod.NewProgress("validating downloads...")
	defer va.Done()

	// always request new manual-url-checksums to avoid potentially reusing existing stale data
	manualUrlChecksums, err := vangoghGetGogChecksums(id, rdx, true)
	if err != nil {
		return err
	}

	// TODO: currently this never returns an error, consider replacing redownload loop with an error
	// and a parameter `no-validation`

	var mismatchedManualUrls []string
	if mismatchedManualUrls, err = vangoghValidateLinks(id, ii, manualUrlFilter, originData, manualUrlChecksums); err != nil {
		return err
	} else if len(mismatchedManualUrls) > 0 {

		// redownload and revalidate any manual-urls that resulted in mismatched checksums

		ii.force = true

		if err = Download(id, ii, nil, mismatchedManualUrls...); err != nil {
			return err
		}

		if _, err = vangoghValidateLinks(id, ii, manualUrlFilter, originData, manualUrlChecksums); err != nil {
			return err
		}
	}

	return nil
}

func vangoghValidateLinks(id string,
	ii *InstallInfo,
	manualUrlFilter []string,
	originData *data.OriginData,
	manualUrlChecksums map[string]string) ([]string, error) {

	vla := nod.NewProgress("validating %s...", id)
	defer vla.Done()

	downloadsDir := camino.GetAbs(vangogh_integration.Downloads)

	downloadTypes := []vangogh_integration.DownloadType{vangogh_integration.Installer}
	if !ii.NoDlcs {
		downloadTypes = append(downloadTypes, vangogh_integration.DLC)
	}

	downloadsList, err := vangogh_integration.FromDetails(originData.GogDetails)
	if err != nil {
		return nil, err
	}

	downloadsList = downloadsList.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLangCodes(ii.LangCode).
		FilterDownloadTypes(downloadTypes...).
		FilterPatches(true)

	if len(downloadsList) == 0 {
		return nil, errors.New("no links are matching operating params")
	}

	vla.TotalInt(len(downloadsList))

	results := make([]ValidationResult, 0, len(downloadsList))

	var mismatchedManualUrls []string

	for _, dl := range downloadsList {
		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}

		var localFilename string
		if localFilename = originData.GogFilenames[dl.ManualUrl]; localFilename == "" {
			continue
		}

		var vr ValidationResult
		vr, err = vangoghValidateLink(id, localFilename, manualUrlChecksums[dl.ManualUrl], downloadsDir)
		if err != nil {
			vla.Error(err)
		}

		if vr == ValResMismatch {
			mismatchedManualUrls = append(mismatchedManualUrls, dl.ManualUrl)
		}

		results = append(results, vr)
	}

	vla.EndWithResult(summarizeValidationResults(results))

	return mismatchedManualUrls, nil
}

func vangoghValidateLink(id string, localFilename string, manualUrlMd5 string, downloadsDir string) (ValidationResult, error) {

	dla := nod.NewProgress(" - %s...", localFilename)
	defer dla.Done()

	absDownloadPath := filepath.Join(downloadsDir, id, localFilename)

	var stat os.FileInfo
	var err error

	if stat, err = os.Stat(absDownloadPath); os.IsNotExist(err) {
		dla.EndWithResult(ValResFileNotFound)
		return ValResFileNotFound, nil
	}

	if manualUrlMd5 == "" {
		dla.EndWithResult(ValResMissingChecksum)
		return ValResMissingChecksum, nil
	}

	dla.Total(uint64(stat.Size()))

	localFile, err := os.Open(absDownloadPath)
	if err != nil {
		return ValResError, err
	}

	h := md5.New()
	if err = dolo.CopyWithProgress(h, localFile, dla); err != nil {
		return ValResError, err
	}

	computedMd5 := fmt.Sprintf("%x", h.Sum(nil))
	if manualUrlMd5 == computedMd5 {
		dla.EndWithResult(ValResValid)
		return ValResValid, nil
	} else {
		dla.EndWithResult(ValResMismatch)
		return ValResMismatch, nil
	}
}

func vangoghGetGogChecksums(id string, rdx redux.Writeable, force bool) (map[string]string, error) {
	rcGogChecksums, err := getProductType(id, vangogh_integration.GogChecksums, rdx, force)
	if err != nil {
		return nil, err
	}
	defer rcGogChecksums.Close()

	var gogChecksums map[string]string
	if err = json.UnmarshalRead(rcGogChecksums, &gogChecksums); err != nil {
		return nil, err
	}

	return gogChecksums, nil
}

func vangoghGetGogFilenames(id string, rdx redux.Writeable, force bool) (map[string]string, error) {
	rcGogFilenames, err := getProductType(id, vangogh_integration.GogFilenames, rdx, force)
	if err != nil {
		return nil, err
	}
	defer rcGogFilenames.Close()

	var gogFilenames map[string]string
	if err = json.UnmarshalRead(rcGogFilenames, &gogFilenames); err != nil {
		return nil, err
	}

	return gogFilenames, nil
}

func vangoghGetGogDetails(id string, rdx redux.Writeable, force bool) (*gog_integration.Details, error) {
	rcDetails, err := getProductType(id, vangogh_integration.GogDetails, rdx, force)
	if err != nil {
		return nil, err
	}
	defer rcDetails.Close()

	return vangogh_integration.UnmarshalDetailsReadCloser(rcDetails)
}

func vangoghGetGogApiProduct(id string, rdx redux.Writeable, force bool) (*gog_integration.ApiProduct, error) {
	rcApiProduct, err := getProductType(id, vangogh_integration.GogApiProducts, rdx, force)
	if err != nil {
		return nil, err
	}
	defer rcApiProduct.Close()

	var apiProduct gog_integration.ApiProduct
	if err = json.UnmarshalRead(rcApiProduct, &apiProduct); err != nil {
		return nil, err
	}

	return &apiProduct, nil
}

func vangoghReduceGogDetails(id string, kvGogDetails kevlar.KeyValues, rdx redux.Writeable) error {

	rcGogDetails, err := kvGogDetails.Get(id)
	if err != nil {
		return err
	}

	defer rcGogDetails.Close()

	det, err := vangogh_integration.UnmarshalDetails(id, kvGogDetails)
	if err != nil {
		return err
	}

	propertyValues := make(map[string][]string)

	reductionProperties := []string{
		vangogh_integration.GogTitleProperty,
		vangogh_integration.GogOperatingSystemsProperty,
	}

	for _, property := range reductionProperties {

		var values []string

		switch property {
		case vangogh_integration.GogTitleProperty:
			values = []string{det.GetTitle()}
		case vangogh_integration.GogOperatingSystemsProperty:
			values, err = det.GetOperatingSystems()
			if err != nil {
				return err
			}
		}

		if len(values) == 1 && values[0] == "" {
			values = nil
		}

		if len(values) > 0 {
			propertyValues[property] = values
		}
	}

	for property, values := range propertyValues {
		if err = rdx.ReplaceValues(property, id, values...); err != nil {
			return err
		}
	}

	return nil
}

func vangoghGetAvailableProducts(rdx redux.Writeable, force bool) ([]vangogh_integration.AvailableProduct, error) {

	rcAvailableProducts, err := getProductType("", vangogh_integration.AvailableProducts, rdx, force)
	if err != nil {
		return nil, err
	}

	defer rcAvailableProducts.Close()

	var availableProducts []vangogh_integration.AvailableProduct
	if err = json.UnmarshalRead(rcAvailableProducts, &availableProducts); err != nil {
		return nil, err
	}

	return availableProducts, nil
}

func vangoghApiProductShortcutAssets(apiProduct *gog_integration.ApiProduct) map[string]string {

	gogAssets := make(map[string]string)

	if apiProduct == nil {
		return gogAssets
	}

	assetImageTypes := []gog_integration.ImageType{
		gog_integration.HorizontalImage,
		gog_integration.VerticalImage,
		gog_integration.Hero,
		gog_integration.Background,
		gog_integration.Logo,
		gog_integration.Icon,
		gog_integration.IconSquare,
	}

	for _, it := range assetImageTypes {

		var getImage func() string

		switch it {
		case gog_integration.HorizontalImage:
			getImage = apiProduct.GetImage
		case gog_integration.VerticalImage:
			getImage = apiProduct.GetVerticalImage
		case gog_integration.Hero:
			getImage = apiProduct.GetHero
		case gog_integration.Background:
			getImage = apiProduct.GetBackground
		case gog_integration.Logo:
			getImage = apiProduct.GetLogo
		case gog_integration.Icon:
			getImage = apiProduct.GetIcon
		case gog_integration.IconSquare:
			getImage = apiProduct.GetIconSquare
		default:
			panic("unknown vangogh asset image type")
		}

		if getImage != nil {
			itp := vangogh_integration.PropertyFromImageType(it)
			gogAssets[itp] = gog_integration.ImageId(getImage())
		}
	}

	return gogAssets
}
