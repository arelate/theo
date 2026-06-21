package cli

import (
	"errors"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

func InstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
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
		OperatingSystem:        operatingSystem,
		LangCode:               langCode,
		DownloadTypes:          downloadTypes,
		Origin:                 data.VangoghOrigin,
		KeepDownloads:          q.Has("keep-downloads"),
		NoSteamShortcut:        q.Has("no-steam-shortcut"),
		NoPresentLaunchOptions: q.Has("no-preset-launch-options"),
		verbose:                q.Has("verbose"),
		force:                  q.Has("force"),
	}

	if q.Has("steam") {
		ii.Origin = data.SteamOrigin
	}

	if q.Has("epic-games") {
		ii.Origin = data.EpicGamesOrigin
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

	vangogh_integration.PrintParams([]string{id},
		[]vangogh_integration.OperatingSystem{ii.OperatingSystem},
		[]string{ii.LangCode},
		ii.DownloadTypes,
		true)

	// don't check existing installations for DLCs, Extras
	if slices.Contains(ii.DownloadTypes, vangogh_integration.Installer) && !ii.force {
		var ok bool
		if ok, err = hasInstallInfo(id, ii, rdx); ok && err == nil {
			ia.EndWithResult("already installed")
			return nil
		} else if err != nil {
			return err
		}
	}

	if err = BackupMetadata(); err != nil {
		return err
	}

	originData, err := originGetData(id, ii, rdx, true)
	if err != nil {
		return err
	}

	if err = Download(id, ii, originData); err != nil {
		return err
	}

	if err = Validate(id, ii); err != nil {
		return err
	}

	if err = osPreInstallActions(id, ii, rdx); err != nil {
		return err
	}

	if err = originInstall(id, ii, originData, rdx); err != nil {
		return err
	}

	if err = originAddSteamShortcut(id, id, ii, originData, rdx); err != nil {
		return err
	}

	if err = originPostInstall(id, ii, originData, rdx); err != nil {
		return err
	}

	if !ii.KeepDownloads {
		if err = RemoveDownloads(id, ii, rdx); err != nil {
			return err
		}
	}

	if err = originPinInstallInfo(id, ii, originData, rdx); err != nil {
		return err
	}

	if !ii.NoPresentLaunchOptions {
		if err = PresetLaunchOptions(id, ii, rdx); err != nil {
			return err
		}
	}

	idInstalledDate := map[string][]string{id: {time.Now().UTC().Format(time.RFC3339)}}
	if err = rdx.BatchReplaceValues(data.InstallDateProperty, idInstalledDate); err != nil {
		return err
	}

	return nil
}

func originPinInstallInfo(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable) error {

	switch ii.Origin {
	case data.EpicGamesOrigin:
		// don't pin EGS DLC install info, as it's already tracked in the pinned main game item install info
		if len(originData.CatalogItem.MainGameItemList) > 0 {
			return nil
		}
	default:
		// do nothing
	}

	return pinInstallInfo(id, ii, rdx)
}

func originAddSteamShortcut(id, forId string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable) error {

	if ii.NoSteamShortcut {
		return nil
	}

	var pda map[steam_grid.Asset]*url.URL
	var lp *logoPosition
	var authToken string
	var err error

	switch ii.Origin {
	case data.VangoghOrigin:
		if originData.ProductDetails != nil {
			pda, err = vangoghShortcutAssets(originData.ProductDetails, rdx)
			if err != nil {
				return err
			}
		}
		lp = defaultLogoPosition()
		if token, ok := rdx.GetLastVal(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty); ok && token != "" {
			authToken = token
		}
	case data.SteamOrigin:
		if originData.AppInfoKv != nil {
			pda, err = steamShortcutAssets(id, originData.AppInfoKv)
			if err != nil {
				return err
			}

			lp, err = steamLogoPosition(id, originData.AppInfoKv)
			if err != nil {
				return err
			}
		}
	case data.EpicGamesOrigin:

		// do not create Steam shortcut for EGS DLCs
		if len(originData.CatalogItem.MainGameItemList) > 0 {
			return nil
		}

		if originData.CatalogItem != nil {
			pda, err = egsCatalogItemAssets(originData.CatalogItem)
			if err != nil {
				return err
			}

			lp = defaultLogoPosition()
		}

	default:
		return ii.Origin.ErrUnsupportedOrigin()
	}

	sgo := &steamGridOptions{
		assets:       pda,
		logoPosition: lp,
		bearerToken:  authToken,
	}

	if forId == "" {
		forId = id
	}

	return addSteamShortcut(forId, ii, rdx, sgo)
}

func originInstall(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable) error {

	switch ii.Origin {
	case data.VangoghOrigin:
		return vangoghUnpackPlace(id, ii, originData, rdx)
	case data.SteamOrigin:
		// do nothing - SteamCMD app update during Download is equivalent to installation
		return nil
	case data.EpicGamesOrigin:
		return egsAssembleValidateChunks(id, ii, originData, rdx)
	default:
		return ii.Origin.ErrUnsupportedOrigin()
	}
}

func originPostInstall(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable) error {

	switch ii.Origin {
	case data.EpicGamesOrigin:
		if err := egsChmodLauncherExe(id, ii, originData, rdx); err != nil {
			return err
		}

		if err := egsInstallDownloadableContent(ii, originData.CatalogItem); err != nil {
			return err
		}

	default:
		// do nothing
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
			return prefixInit(id, ii.Origin, rdx, ii.verbose)
		default:
			return nil
		}
	default:
		return nil
	}
}

func originOsInstalledPath(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {

	switch ii.Origin {
	case data.VangoghOrigin:

		if err := rdx.MustHave(vangogh_integration.GogBundleNameProperty); err != nil {
			return "", err
		}

		installedAppsDir := data.Pwd.AbsDirPath(data.InstalledApps)

		osLangInstalledAppsDir := filepath.Join(installedAppsDir, data.OsLangCode(ii.OperatingSystem, ii.LangCode))

		title, err := data.GetTitleProperty(id, rdx)
		if err != nil {
			return "", err
		}

		appInstalledPath := pathways.Sanitize(title)

		switch ii.OperatingSystem {
		case vangogh_integration.MacOS:
			if bundleName, sure := rdx.GetLastVal(vangogh_integration.GogBundleNameProperty, id); sure && bundleName != "" {
				appInstalledPath = filepath.Join(appInstalledPath, bundleName)
			}
		default:
			// do nothing
		}

		return filepath.Join(osLangInstalledAppsDir, appInstalledPath), nil
	case data.SteamOrigin:
		if steamAppInstallDir, err := data.AbsSteamAppInstallDir(id, ii.OperatingSystem, rdx); err == nil {
			return steamAppInstallDir, nil
		} else {
			return "", err
		}
	case data.EpicGamesOrigin:
		egsAppsDir := data.Pwd.AbsDirPath(data.EgsApps)

		osEgsAppsDir := filepath.Join(egsAppsDir, ii.OperatingSystem.String())

		var appTitlePath string

		// for EGS DLCs - use main game item appName to set the installation directory
		if requiresGame, ok := rdx.GetLastVal(vangogh_integration.EgsMainGameProperty, id); ok && requiresGame != "" {
			id = requiresGame
		}

		if title, sure := rdx.GetLastVal(vangogh_integration.EgsTitleProperty, id); sure && title != "" {
			appTitlePath = pathways.Sanitize(title)
		} else {
			return "", errors.New("product title not defined for: " + id)
		}

		return filepath.Join(osEgsAppsDir, appTitlePath), nil
	default:
		return "", ii.Origin.ErrUnsupportedOrigin()
	}
}
