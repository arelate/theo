package cli

import (
	"errors"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func getSteamAppInfoKv(steamAppId string, rdx redux.Writeable, force bool) (steam_vdf.ValveDataFile, error) {

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

	if err = fetchSteamAppInfo(steamAppId, username, kvSteamAppInfo, rdx, force); err != nil {
		return nil, err
	}

	appInfoRc, err := kvSteamAppInfo.Get(steamAppId)
	if err != nil {
		return nil, err
	}
	defer appInfoRc.Close()

	return steam_vdf.ReadText(appInfoRc)
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

		var appInfoKv steam_vdf.ValveDataFile
		appInfoKv, err = steam_vdf.ReadText(strings.NewReader(printedAppInfo))
		if err != nil {
			return err
		}

		if err = steamReduceAppInfo(steamAppId, appInfoKv, rdx); err != nil {
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

func steamReduceAppInfo(steamAppId string, appInfoKv steam_vdf.ValveDataFile, rdx redux.Writeable) error {

	if err := rdx.MustHave(vangogh_integration.TitleProperty); err != nil {
		return err
	}

	var appInfoName string
	if ain, ok := appInfoKv.Val(steamAppId, "common", "name"); ok {
		appInfoName = ain
	}

	if err := rdx.ReplaceValues(vangogh_integration.TitleProperty, steamAppId, appInfoName); err != nil {
		return err
	}

	return nil
}

func steamShortcutAssets(steamAppId string, appInfoKv steam_vdf.ValveDataFile) (map[steam_grid.Asset]*url.URL, error) {

	shortcutAssets := make(map[steam_grid.Asset]*url.URL)

	for _, asset := range steam_grid.ShortcutAssets {

		var imageId string
		var err error

		switch asset {
		case steam_grid.Header:

			if imageId, err = steamAppInfoAsset(appInfoKv,
				[]string{steamAppId, "common", "library_assets_full", "library_header", "image", "english"},
				[]string{steamAppId, "common", "header_image", "english"}...); err != nil {
				return nil, err
			}

		case steam_grid.LibraryCapsule:

			if imageId, err = steamAppInfoAsset(appInfoKv,
				[]string{steamAppId, "common", "library_assets_full", "library_capsule", "image", "english"}); err != nil {
				return nil, err
			}

		case steam_grid.LibraryHero:

			if imageId, err = steamAppInfoAsset(appInfoKv,
				[]string{steamAppId, "common", "library_assets_full", "library_hero", "image", "english"}); err != nil {
				return nil, err
			}

		case steam_grid.LibraryLogo:

			if imageId, err = steamAppInfoAsset(appInfoKv,
				[]string{steamAppId, "common", "library_assets_full", "library_logo", "image", "english"}); err != nil {
				return nil, err
			}

		case steam_grid.ClientIcon:

			if imageId, err = steamAppInfoAsset(appInfoKv,
				[]string{steamAppId, "common", "clienticon"}); err != nil {
				return nil, err
			}

		default:
			return nil, errors.New("unexpected shortcut asset " + asset.String())
		}

		if imageId == "" {
			if defaultImageId, ok := steam_grid.DefaultAssetsFilenames[asset]; ok {
				imageId = defaultImageId
			}
		}

		if imageId != "" {
			shortcutAssets[asset] = steam_grid.AssetUrl(steamAppId, imageId, asset)
		}
	}

	return shortcutAssets, nil
}

func steamAppInfoAsset(appInfoKv steam_vdf.ValveDataFile, preferredPath []string, fallbackPath ...string) (string, error) {

	if preferredAsset, ok := appInfoKv.Val(preferredPath...); ok {
		return preferredAsset, nil
	} else {

		if len(fallbackPath) > 0 {

			if fallbackAsset, sure := appInfoKv.Val(fallbackPath...); sure {
				return fallbackAsset, nil
			}
		}
	}

	return "", nil
}

func steamLogoPosition(steamAppId string, appInfoKv steam_vdf.ValveDataFile) (*logoPosition, error) {

	shortcutLogoPosition := new(logoPosition)

	if pinnedPosition, err := steamAppInfoAsset(appInfoKv,
		[]string{steamAppId, "common", "library_assets_full", "library_logo", "logo_position", "pinned_position"}); err == nil {
		shortcutLogoPosition.PinnedPosition = pinnedPosition
	} else {
		return nil, err
	}

	if wps, err := steamAppInfoAsset(appInfoKv,
		[]string{steamAppId, "common", "library_assets_full", "library_logo", "logo_position", "width_pct"}); err == nil {

		var wpf float64
		if wpf, err = strconv.ParseFloat(wps, 64); err == nil {
			shortcutLogoPosition.WidthPct = wpf
		} else {
			return nil, err
		}

	} else {
		return nil, err
	}

	if hps, err := steamAppInfoAsset(appInfoKv,
		[]string{steamAppId, "common", "library_assets_full", "library_logo", "logo_position", "height_pct"}); err == nil {

		var hpf float64
		if hpf, err = strconv.ParseFloat(hps, 64); err == nil {
			shortcutLogoPosition.HeightPct = hpf
		} else {
			return nil, err
		}

	} else {
		return nil, err
	}

	if shortcutLogoPosition.PinnedPosition == "" {
		shortcutLogoPosition = defaultLogoPosition()
	}

	return shortcutLogoPosition, nil
}

func steamAppInfoVersion(steamAppId string, appInfoKv steam_vdf.ValveDataFile) (string, error) {

	if buildId, ok := appInfoKv.Val(steamAppId, "depots", "branches", "public", "buildid"); ok && buildId != "" {
		return buildId, nil
	}
	return "", errors.New("appinfo is missing depots/branches/public/buildid")
}

func steamAppInfoTimeUpdated(steamAppId string, appInfoKv steam_vdf.ValveDataFile) (time.Time, error) {
	if timeUpdated, ok := appInfoKv.Val(steamAppId, "depots", "branches", "public", "timeupdated"); ok && timeUpdated != "" {
		if tuu, err := strconv.ParseInt(timeUpdated, 10, 64); err == nil {

			return time.Unix(tuu, 0), nil
		} else {
			return time.Time{}, err
		}
	}
	return time.Time{}, nil
}

func steamAppInfoSize(steamAppId string, operatingSystem vangogh_integration.OperatingSystem, appInfoKv steam_vdf.ValveDataFile) (int64, error) {

	depotsKv, err := appInfoKv.At(steamAppId, "depots")
	if errors.Is(err, steam_vdf.ErrVdfKeyNotFound) {
		// depots key not present stop reducing
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	var estimatedBytes int64

	for _, depotKv := range depotsKv.Values {

		// skip key values that don't have Values, e.g. "baselanguages
		if len(depotKv.Values) == 0 {
			continue
		}

		// skip depots that have depotfromapp or sharedinstall set (Redistributables depot)
		if depotFromApp, ok := depotsKv.Values.Val("depotfromapp"); ok && depotFromApp != "" {
			continue
		}
		if sharedInstall, ok := depotKv.Values.Val("sharedinstall"); ok && sharedInstall != "" {
			continue
		}

		if depotOsList, ok := depotKv.Values.Val("config", "oslist"); ok && depotOsList != "" {

			depotOperatingSystems := vangogh_integration.ParseManyOperatingSystems(strings.Split(depotOsList, ","))
			if !slices.Contains(depotOperatingSystems, operatingSystem) {
				continue
			}

			if sizeStr, sure := depotKv.Values.Val("manifests", "public", "size"); sure && sizeStr != "" {
				var sizeInt int64
				if sizeInt, err = strconv.ParseInt(sizeStr, 10, 64); err == nil {
					estimatedBytes += sizeInt
				} else {
					return 0, err
				}
			}

		} else {
			continue
		}
	}

	return estimatedBytes, nil
}
