package cli

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/arelate/southern_light/steam_appinfo"
	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func getSteamAppInfo(steamAppId string, ii *InstallInfo, rdx redux.Writeable) (*steam_appinfo.AppInfo, error) {

	steamAppInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)

	kvSteamAppInfo, err := kevlar.New(steamAppInfoDir, steam_vdf.Ext)
	if err != nil {
		return nil, err
	}

	var username string
	if un, ok := rdx.GetLastVal(data.SteamUsernameProperty, data.SteamUsernameProperty); ok && un != "" {
		username = un
	} else {
		return nil, errors.New("cannot resolve Steam username")
	}

	if err = fetchSteamAppInfo(steamAppId, username, kvSteamAppInfo, rdx, ii.force); err != nil {
		return nil, err
	}

	appInfoRc, err := kvSteamAppInfo.Get(steamAppId)
	if err != nil {
		return nil, err
	}
	defer appInfoRc.Close()

	appInfoKeyValues, err := steam_vdf.ReadText(appInfoRc)
	if err != nil {
		return nil, err
	}

	return steam_appinfo.AppInfoVdf(appInfoKeyValues)
}

func fetchSteamAppInfo(steamAppId string, username string, kvSteamAppInfo kevlar.KeyValues, rdx redux.Writeable, force bool) error {

	scaia := nod.Begin(" getting Steam appinfo for %s...", steamAppId)
	defer scaia.Done()

	if kvSteamAppInfo.Has(steamAppId) && !force {
		scaia.EndWithResult("already exist")
		return nil
	}

	absSteamCmdPath, err := data.AbsSteamCmdBinPath(data.CurrentOs())
	if err != nil {
		return err
	}

	printedAppInfo, err := steamcmd.AppInfoPrint(absSteamCmdPath, steamAppId)
	if err != nil {
		return err
	}

	if strings.Replace(printedAppInfo, steamAppId, "", 1) == "\"\"{}" { // empty appinfo for steamAppId: "steamAppId"{}
		return errors.New("empty appinfo for " + steamAppId)
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

		if err = steamReduceAppInfo(appInfo, rdx); err != nil {
			return err
		}
	}

	return nil
}

func steamUpdateApp(steamAppId string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) error {

	var steamAppName string
	if san, ok := rdx.GetLastVal(vangogh_integration.TitleProperty, steamAppId); ok && san != "" {
		steamAppName = san
	} else {
		return errors.New("cannot resolve Steam app title")
	}

	var steamUsername string
	if sun, ok := rdx.GetLastVal(data.SteamUsernameProperty, data.SteamUsernameProperty); ok && sun != "" {
		steamUsername = sun
	} else {
		return errors.New("cannot resolve Steam username")
	}

	scaua := nod.Begin("updating and verifying %s (%s) for %s with SteamCMD, please wait...", steamAppName, steamAppId, operatingSystem)
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

	absSteamCmdPath, err := data.AbsSteamCmdBinPath(data.CurrentOs())
	if err != nil {
		return err
	}

	return steamcmd.AppUpdate(absSteamCmdPath, steamAppId, operatingSystem, steamAppInstallDir, steamUsername)
}

func steamReduceAppInfo(appInfo *steam_appinfo.AppInfo, rdx redux.Writeable) error {

	if err := rdx.MustHave(vangogh_integration.TitleProperty); err != nil {
		return err
	}

	if err := rdx.ReplaceValues(vangogh_integration.TitleProperty, appInfo.AppId, appInfo.Common.Name); err != nil {
		return err
	}

	return nil
}

func steamShortcutAssets(appInfo *steam_appinfo.AppInfo) (map[steam_grid.Asset]*url.URL, *logoPosition, error) {

	shortcutAssets := make(map[steam_grid.Asset]*url.URL)

	for _, asset := range steam_grid.ShortcutAssets {

		var imageId string
		switch asset {
		case steam_grid.Header:
			if appInfo.Common.LibraryAssetsFull != nil &&
				appInfo.Common.LibraryAssetsFull.LibraryHeader != nil {
				imageId = appInfo.Common.LibraryAssetsFull.LibraryHeader.DefaultImage()
			} else if dh := appInfo.Common.DefaultHeaderImage(); dh != "" {
				imageId = dh
			}
		case steam_grid.LibraryCapsule:
			if appInfo.Common.LibraryAssetsFull != nil &&
				appInfo.Common.LibraryAssetsFull.LibraryCapsule != nil {
				imageId = appInfo.Common.LibraryAssetsFull.LibraryCapsule.DefaultImage()
			}
		case steam_grid.LibraryHero:
			if appInfo.Common.LibraryAssetsFull != nil &&
				appInfo.Common.LibraryAssetsFull.LibraryHero != nil {
				imageId = appInfo.Common.LibraryAssetsFull.LibraryHero.DefaultImage()
			}
		case steam_grid.LibraryLogo:
			if appInfo.Common.LibraryAssetsFull != nil &&
				appInfo.Common.LibraryAssetsFull.LibraryLogo != nil {
				imageId = appInfo.Common.LibraryAssetsFull.LibraryLogo.DefaultImage()
			}
		case steam_grid.ClientIcon:
			imageId = appInfo.Common.ClientIcon
		default:
			return nil, nil, errors.New("unexpected shortcut asset " + asset.String())
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

	var shortcutLogoPosition *logoPosition
	if appInfo.Common.LibraryAssetsFull != nil {
		shortcutLogoPosition = new(logoPosition{
			PinnedPosition: appInfo.Common.LibraryAssetsFull.LibraryLogo.PinnedPosition,
			WidthPct:       appInfo.Common.LibraryAssetsFull.LibraryLogo.WidthPct,
			HeightPct:      appInfo.Common.LibraryAssetsFull.LibraryLogo.HeighPct,
		})
	} else {
		shortcutLogoPosition = defaultLogoPosition()
	}

	return shortcutAssets, shortcutLogoPosition, nil
}
