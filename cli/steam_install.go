package cli

import (
	"errors"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/arelate/southern_light/steam_appinfo"
	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func SteamInstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        defaultLangCode,
		DownloadTypes:   []vangogh_integration.DownloadType{vangogh_integration.Installer},
		SteamInstall:    true,
		NoSteamShortcut: q.Has("no-steam-shortcut"),
		reveal:          q.Has("reveal"),
		verbose:         q.Has("verbose"),
		force:           q.Has("force"),
	}

	return SteamInstall(id, ii)
}

func SteamInstall(steamAppId string, ii *InstallInfo) error {

	sia := nod.Begin("installing Steam %s for %s...", steamAppId, ii.OperatingSystem)
	defer sia.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	steamAppInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)
	kvSteamAppInfo, err := kevlar.New(steamAppInfoDir, steam_vdf.Ext)
	if err != nil {
		return err
	}

	var username string
	if un, ok := rdx.GetLastVal(data.SteamUsernameProperty, data.SteamUsernameProperty); ok && un != "" {
		username = un
	}

	if err = BackupMetadata(); err != nil {
		return err
	}

	if err = getSteamAppInfo(steamAppId, username, kvSteamAppInfo, rdx, ii.force); err != nil {
		return err
	}

	appInfoRc, err := kvSteamAppInfo.Get(steamAppId)
	if err != nil {
		return err
	}
	defer appInfoRc.Close()

	appInfoKeyValues, err := steam_vdf.ReadText(appInfoRc)
	if err != nil {
		return err
	}

	appInfo, err := steam_appinfo.AppInfoVdf(appInfoKeyValues)
	if err != nil {
		return err
	}

	productDetails := steamAppInfoProductDetails(appInfo)

	if err = resolveInstallInfo(steamAppId, ii, productDetails, rdx, currentOsThenWindows); err != nil {
		return err
	}

	printInstallInfoParams(ii, true, steamAppId)

	if slices.Contains(ii.DownloadTypes, vangogh_integration.Installer) && !ii.force {

		if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, steamAppId); ok {

			var installInfo *InstallInfo
			installInfo, _, err = matchInstallInfo(ii, installedInfoLines...)
			if err != nil {
				return err
			}

			if installInfo != nil {
				sia.EndWithResult("Steam appId %s is already installed for %s", steamAppId, ii.OperatingSystem)
				return nil
			}
		}
	}

	if ii.OperatingSystem == vangogh_integration.Windows && data.CurrentOs() != vangogh_integration.Windows {
		if err = steamPrefixInit(steamAppId, ii.verbose); err != nil {
			return err
		}
	}

	if err = steamUpdateApp(steamAppId, appInfo.Common.Name, username, ii.OperatingSystem, rdx); err != nil {
		return err
	}

	steamAppInstallDir, err := data.AbsSteamAppInstallDir(steamAppId, ii.OperatingSystem, rdx)
	if err != nil {
		return err
	}

	shortcutAssets, err := appInfoShortcutAssets(appInfo)
	if err != nil {
		return err
	}

	shortcutLogoPosition := &logoPosition{
		PinnedPosition: appInfo.Common.LibraryAssetsFull.LibraryLogo.PinnedPosition,
		WidthPct:       appInfo.Common.LibraryAssetsFull.LibraryLogo.WidthPct,
		HeightPct:      appInfo.Common.LibraryAssetsFull.LibraryLogo.HeighPct,
	}

	sgo := &steamGridOptions{
		additions:    []string{steamAppId},
		steamRun:     true,
		assets:       shortcutAssets,
		name:         appInfo.Common.Name,
		installDir:   steamAppInstallDir,
		logoPosition: shortcutLogoPosition,
	}

	if err = SteamShortcut(ii, sgo); err != nil {
		return err
	}

	if err = pinInstallInfo(steamAppId, ii, rdx); err != nil {
		return err
	}

	return nil
}

func getSteamAppInfo(steamAppId string, username string, kvSteamAppInfo kevlar.KeyValues, rdx redux.Writeable, force bool) error {

	scaia := nod.Begin(" getting Steam appinfo for %s...", steamAppId)
	defer scaia.Done()

	if kvSteamAppInfo.Has(steamAppId) && !force {
		scaia.EndWithResult("already exist")
		return nil
	}

	printedAppInfo, err := steamCmdAppInfoPrint(steamAppId, username)
	if err != nil {
		return err
	}

	if err = kvSteamAppInfo.Set(steamAppId, strings.NewReader(printedAppInfo)); err != nil {
		return err
	}

	if !rdx.HasKey(vangogh_integration.TitleProperty, steamAppId) || force {

		var appInfoKeyValues []*steam_vdf.KeyValues
		appInfoKeyValues, err = steam_vdf.ReadText(strings.NewReader(printedAppInfo))
		if err != nil {
			return err
		}

		var appInfo *steam_appinfo.AppInfo
		appInfo, err = steam_appinfo.AppInfoVdf(appInfoKeyValues)
		if err != nil {
			return err
		}

		if err = reduceSteamAppInfo(appInfo, rdx); err != nil {
			return err
		}
	}

	return nil
}

func steamUpdateApp(steamAppId, name string, username string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) error {

	scaua := nod.Begin("updating and verifying %s (%s) for %s with SteamCMD, please wait...", name, steamAppId, operatingSystem)
	defer scaua.Done()

	steamAppInstallDir, err := data.AbsSteamAppInstallDir(steamAppId, operatingSystem, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(steamAppInstallDir); os.IsNotExist(err) {
		if err = os.MkdirAll(steamAppInstallDir, 0755); err != nil {
			return err
		}
	}

	return steamCmdAppUpdate(steamAppId, operatingSystem, steamAppInstallDir, username)
}

func steamAppInfoProductDetails(appInfo *steam_appinfo.AppInfo) *vangogh_integration.ProductDetails {

	var operatingSystems []vangogh_integration.OperatingSystem
	if appInfo.Common.OsList != "" {
		operatingSystems = vangogh_integration.ParseManyOperatingSystems(strings.Split(appInfo.Common.OsList, ","))
	} else {
		operatingSystems = append(operatingSystems, vangogh_integration.Windows)
	}

	productDetails := &vangogh_integration.ProductDetails{
		SteamAppId:       appInfo.AppId,
		Title:            appInfo.Common.Name,
		ProductType:      vangogh_integration.GameProductType,
		OperatingSystems: operatingSystems,
		Developers:       []string{appInfo.Extended.Developer},
		Publishers:       []string{appInfo.Extended.Publisher},
	}

	return productDetails
}

func reduceSteamAppInfo(appInfo *steam_appinfo.AppInfo, rdx redux.Writeable) error {

	if err := rdx.MustHave(vangogh_integration.TitleProperty); err != nil {
		return err
	}

	if err := rdx.ReplaceValues(vangogh_integration.TitleProperty, appInfo.AppId, appInfo.Common.Name); err != nil {
		return err
	}

	return nil
}

func appInfoShortcutAssets(appInfo *steam_appinfo.AppInfo) (map[steam_grid.Asset]*url.URL, error) {

	shortcutAssets := make(map[steam_grid.Asset]*url.URL)

	for _, asset := range steam_grid.ShortcutAssets {

		var imageId string
		switch asset {
		case steam_grid.Header:
			if appInfo.Common.LibraryAssetsFull.LibraryHeader != nil {
				imageId = appInfo.Common.LibraryAssetsFull.LibraryHeader.DefaultImage()
			} else if dh := appInfo.Common.DefaultHeaderImage(); dh != "" {
				imageId = dh
			}
		case steam_grid.LibraryCapsule:
			if appInfo.Common.LibraryAssetsFull.LibraryCapsule != nil {
				imageId = appInfo.Common.LibraryAssetsFull.LibraryCapsule.DefaultImage()
			}
		case steam_grid.LibraryHero:
			if appInfo.Common.LibraryAssetsFull.LibraryHero != nil {
				imageId = appInfo.Common.LibraryAssetsFull.LibraryHero.DefaultImage()
			}
		case steam_grid.LibraryLogo:
			if appInfo.Common.LibraryAssetsFull.LibraryLogo != nil {
				imageId = appInfo.Common.LibraryAssetsFull.LibraryLogo.DefaultImage()
			}
		case steam_grid.ClientIcon:
			imageId = appInfo.Common.ClientIcon
		default:
			return nil, errors.New("unexpected shortcut asset " + asset.String())
		}

		if imageId == "" {
			if defaultImageId, ok := steam_grid.DefaultAssetsFilenames[asset]; ok {
				imageId = defaultImageId
			}
		}

		if imageId != "" {
			shortcutAssets[asset] = steam_grid.AssetUrl(appInfo.AppId, imageId, asset)
		}
	}

	return shortcutAssets, nil
}
