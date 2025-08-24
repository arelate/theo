package cli

import (
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
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

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	printInstallInfoParams(ii, true, id)

	productDetails, err := getProductDetails(id, rdx, ii.force)
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

	if err = resolveInstallInfo(id, ii, rdx, currentOsThenWindows); err != nil {
		return err
	}

	ii.AddProductDetails(productDetails)

	// don't check existing installations for DLCs, Extras
	if slices.Contains(ii.DownloadTypes, vangogh_integration.Installer) && !ii.force {

		if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

			installInfo, err := matchInstallInfo(ii, installedInfoLines...)
			if err != nil {
				return err
			}

			if installInfo != nil {
				ia.EndWithResult("product %s is already installed", id)
				return nil
			} else {
				return err
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

	if err = osInstallProduct(id, ii, productDetails, rdx); err != nil {
		return err
	}

	if !ii.NoSteamShortcut {
		if err = addSteamShortcut(id, ii.OperatingSystem, ii.LangCode, rdx, ii.force); err != nil {
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

func osInstallProduct(id string, ii *InstallInfo, productDetails *vangogh_integration.ProductDetails, rdx redux.Writeable) error {

	start := time.Now().UTC().Unix()

	coipa := nod.Begin("installing %s, %s, %s...", id, ii.OperatingSystem, ii.LangCode)
	defer coipa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		coipa.EndWithResult("no links are matching install params")
		return nil
	}

	for _, dl := range dls {
		if dl.Type == vangogh_integration.DLC {
			ii.DownloadableContent = append(ii.DownloadableContent, dl.Name)
		}
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	if err = hasFreeSpaceForProduct(productDetails, installedAppsDir, ii, nil); err != nil {
		return err
	}

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:

		if err = macOsInstallProduct(id, dls, rdx, ii.force); err != nil {
			return err
		}

	case vangogh_integration.Linux:

		if err = linuxInstallProduct(id, dls, rdx); err != nil {
			return err
		}

	case vangogh_integration.Windows:

		switch data.CurrentOs() {
		case vangogh_integration.Windows:

			if err = windowsInstallProduct(id, dls, rdx, ii.force); err != nil {
				return err
			}

		default:

			if err = prefixInit(id, ii.LangCode, rdx, ii.verbose); err != nil {
				return err
			}

			if err = prefixInstallProduct(id, dls, ii, rdx); err != nil {
				return err
			}

			if err = prefixCreateInventory(id, ii.LangCode, rdx, start); err != nil {
				return err
			}

			if err = prefixDefaultEnv(id, ii.LangCode, rdx); err != nil {
				return err
			}

		}
	default:
		return ii.OperatingSystem.ErrUnsupported()
	}

	return nil
}
