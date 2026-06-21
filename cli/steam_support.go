package cli

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/steam_integration"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func steamGetAppInfoKv(steamAppId string, rdx redux.Writeable, force bool) (steam_vdf.ValveDataFile, error) {

	steamAppInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)

	kvSteamAppInfo, err := kevlar.New(steamAppInfoDir, steam_vdf.Ext)
	if err != nil {
		return nil, err
	}

	if err = steamFetchAppInfo(steamAppId, kvSteamAppInfo, rdx, force); err != nil {
		return nil, err
	}

	appInfoRc, err := kvSteamAppInfo.Get(steamAppId)
	if err != nil {
		return nil, err
	}
	defer appInfoRc.Close()

	return steam_vdf.ReadText(appInfoRc)
}

func steamFetchAppInfo(steamAppId string, kvSteamAppInfo kevlar.KeyValues, rdx redux.Writeable, force bool) error {

	scaia := nod.Begin(" getting Steam appinfo for %s...", steamAppId)
	defer scaia.Done()

	if kvSteamAppInfo.Has(steamAppId) && !force {
		scaia.EndWithResult("read local")
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

	if !rdx.HasKey(vangogh_integration.SteamTitleProperty, steamAppId) || force {

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
	if san, ok := rdx.GetLastVal(vangogh_integration.SteamTitleProperty, steamAppId); ok && san != "" {
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

	return steamcmd.AppUpdate(absSteamCmdPath, steamAppId, operatingSystem, steamAppInstallDir, steamUsername, false)
}

func steamValidateApp(steamAppId string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) error {

	var steamAppName string
	if san, ok := rdx.GetLastVal(vangogh_integration.SteamTitleProperty, steamAppId); ok && san != "" {
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

	return steamcmd.AppUpdate(absSteamCmdPath, steamAppId, operatingSystem, steamAppInstallDir, steamUsername, true)
}

func steamReduceAppInfo(steamAppId string, appInfoKv steam_vdf.ValveDataFile, rdx redux.Writeable) error {

	if err := rdx.MustHave(vangogh_integration.SteamTitleProperty); err != nil {
		return err
	}

	var appInfoName string
	if ain, ok := appInfoKv.Val(steamAppId, "common", "name"); ok {
		appInfoName = ain
	}

	if err := rdx.ReplaceValues(vangogh_integration.SteamTitleProperty, steamAppId, appInfoName); err != nil {
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

	shortcutLogoPosition := defaultLogoPosition()

	if pinnedPosition, err := steamAppInfoAsset(appInfoKv,
		[]string{steamAppId, "common", "library_assets_full", "library_logo", "logo_position", "pinned_position"}); err == nil {
		shortcutLogoPosition.PinnedPosition = pinnedPosition
	} else {
		return nil, err
	}

	if wps, err := steamAppInfoAsset(appInfoKv,
		[]string{steamAppId, "common", "library_assets_full", "library_logo", "logo_position", "width_pct"}); err == nil && wps != "" {

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
		[]string{steamAppId, "common", "library_assets_full", "library_logo", "logo_position", "height_pct"}); err == nil && hps != "" {

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

		if language, ok := depotsKv.Values.Val("config", "language"); ok && language != "english" {
			continue
		}

		var depotOperatingSystems []vangogh_integration.OperatingSystem
		if depotOsList, ok := depotKv.Values.Val("config", "oslist"); ok && depotOsList != "" {
			depotOperatingSystems = vangogh_integration.ParseManyOperatingSystems(strings.Split(depotOsList, ","))
		} else {
			depotOperatingSystems = []vangogh_integration.OperatingSystem{vangogh_integration.Windows}
		}

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

	}

	return estimatedBytes, nil
}

func steamSetupConnection(username string, rdx redux.Writeable, reset bool) error {

	ssca := nod.Begin("connecting to Steam...")
	defer ssca.Done()

	if err := rdx.MustHave(data.SteamProperties()...); err != nil {
		return err
	}

	if reset {
		if err := steamResetConnection(rdx); err != nil {
			return err
		}
	}

	switch username {
	case "":
		if un, ok := rdx.GetLastVal(data.SteamUsernameProperty, data.SteamUsernameProperty); ok && un != "" {
			username = un
		} else {
			return errors.New("please provide Steam username")
		}
	default:
		if err := rdx.ReplaceValues(data.SteamUsernameProperty, data.SteamUsernameProperty, username); err != nil {
			return err
		}
	}

	absSteamCmdPath, err := data.AbsSteamCmdBinPath(data.CurrentOs())
	if err != nil {
		return err
	}

	return steamcmd.Login(absSteamCmdPath, username)
}

func steamResetConnection(rdx redux.Writeable) error {
	rsca := nod.Begin("resetting Steam connection...")
	defer rsca.Done()

	if err := rdx.CutKeys(data.SteamUsernameProperty, data.SteamUsernameProperty); err != nil {
		return err
	}

	absSteamCmdPath, err := data.AbsSteamCmdBinPath(data.CurrentOs())
	if err != nil {
		return err
	}

	return steamcmd.Logout(absSteamCmdPath)
}

func steamDownloadData(steamAppId string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {
	steamAppsDir := data.Pwd.AbsDirPath(data.SteamApps)

	if err := originHasFreeSpace(steamAppId, steamAppsDir, ii, originData); err != nil {
		return err
	}

	return steamUpdateApp(steamAppId, ii.OperatingSystem, rdx)
}

func steamGetExecTask(steamAppId string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable, et *execTask) (*execTask, error) {

	var err error

	switch et.task {
	case "":
		et, err = steamDefaultTask(steamAppId, originData.AppInfoKv, ii, rdx)
		if err != nil {
			return nil, err
		}
	default:
		et, err = steamNamedTask(steamAppId, et.task, originData.AppInfoKv, ii, rdx)
		if err != nil {
			return nil, err
		}
	}

	return et, nil
}

func steamGetLaunchConfigs(steamAppId string, appInfoKv steam_vdf.ValveDataFile) ([]*steam_integration.LaunchConfig, error) {

	appInfoClKv, err := appInfoKv.At(steamAppId, "config", "launch")
	if err != nil {
		return nil, err
	}

	launchConfigs := make([]*steam_integration.LaunchConfig, 0, len(appInfoClKv.Values))

	for _, lcKv := range appInfoClKv.Values {

		lc := new(steam_integration.LaunchConfig)

		if lcExe, ok := lcKv.Values.Val("executable"); ok {
			lc.Executable = lcExe
		}

		if lcArgs, ok := lcKv.Values.Val("arguments"); ok {
			lc.Arguments = lcArgs
		}

		if lcWd, ok := lcKv.Values.Val("workingdir"); ok {
			lc.WorkingDir = lcWd
		}

		if lcDesc, ok := lcKv.Values.Val("description"); ok && lcDesc != "" {
			lc.Description = lcDesc
		}

		if lcType, ok := lcKv.Values.Val("type"); ok {
			lc.Type = lcType
		}

		if lol, ok := lcKv.Values.Val("config", "oslist"); ok {
			lc.OsList = lol
		}

		if loa, ok := lcKv.Values.Val("config", "osarch"); ok {
			lc.OsArch = loa
		}

		if lbk, ok := lcKv.Values.Val("config", "BetaKey"); ok {
			lc.BetaKey = lbk
		}

		launchConfigs = append(launchConfigs, lc)
	}

	return launchConfigs, nil
}

func steamDefaultTask(steamAppId string, appInfoKv steam_vdf.ValveDataFile, ii *InstallInfo, rdx redux.Readable) (*execTask, error) {

	steamLaunchConfigs, err := steamGetLaunchConfigs(steamAppId, appInfoKv)
	if err != nil {
		return nil, err
	}

	steamAppInstallDir, err := data.AbsSteamAppInstallDir(steamAppId, ii.OperatingSystem, rdx)
	if err != nil {
		return nil, err
	}

	absPrefixDir, err := data.AbsPrefixDir(steamAppId, ii.Origin, rdx)
	if err != nil {
		return nil, err
	}

	var appInfoName string
	if ain, ok := appInfoKv.Val(steamAppId, "common", "name"); ok {
		appInfoName = ain
	}

	et := new(execTask)

	for _, slc := range steamLaunchConfigs {

		var lcOperatingSystem vangogh_integration.OperatingSystem

		if slc.OsList != "" {
			osList := vangogh_integration.ParseManyOperatingSystems(strings.Split(slc.OsList, ","))
			switch len(osList) {
			case 0:
				lcOperatingSystem = vangogh_integration.Windows
			case 1:
				lcOperatingSystem = osList[0]
			default:
				return nil, errors.New("more than one steam launch config found for " + steamAppId)
			}
		} else {
			lcOperatingSystem = vangogh_integration.Windows
		}

		if lcOperatingSystem != ii.OperatingSystem ||
			slc.Executable == "" ||
			slc.OsArch == "32" {
			continue
		}

		et.exe = filepath.Join(steamAppInstallDir, windowsToNixPath(slc.Executable))
		et.workDir = filepath.Join(steamAppInstallDir, windowsToNixPath(slc.WorkingDir))
		et.prefix = absPrefixDir
		et.title = slc.Description
		if et.title == "" {
			et.title = appInfoName
		}

		return et, nil
	}

	return nil, errors.New("cannot determine default steam launch config for " + steamAppId)
}

func steamNamedTask(steamAppId, task string, appInfoKv steam_vdf.ValveDataFile, ii *InstallInfo, rdx redux.Readable) (*execTask, error) {

	steamLaunchConfigs, err := steamGetLaunchConfigs(steamAppId, appInfoKv)
	if err != nil {
		return nil, err
	}

	steamAppInstallDir, err := data.AbsSteamAppInstallDir(steamAppId, ii.OperatingSystem, rdx)
	if err != nil {
		return nil, err
	}

	absPrefixDir, err := data.AbsPrefixDir(steamAppId, ii.Origin, rdx)
	if err != nil {
		return nil, err
	}

	var appInfoName string
	if ain, ok := appInfoKv.Val(steamAppId, "common", "name"); ok {
		appInfoName = ain
	}

	et := new(execTask)

	for _, slc := range steamLaunchConfigs {

		var lcOperatingSystem vangogh_integration.OperatingSystem

		if slc.OsList != "" {
			osList := vangogh_integration.ParseManyOperatingSystems(strings.Split(slc.OsList, ","))
			switch len(osList) {
			case 0:
				lcOperatingSystem = vangogh_integration.Windows
			case 1:
				lcOperatingSystem = osList[0]
			default:
				return nil, errors.New("more than one steam launch config found for " + steamAppId)
			}
		} else {
			lcOperatingSystem = vangogh_integration.Windows
		}

		if lcOperatingSystem != ii.OperatingSystem ||
			slc.Executable == "" ||
			slc.OsArch == "32" ||
			slc.Description != task {
			continue
		}

		et.exe = filepath.Join(steamAppInstallDir, windowsToNixPath(slc.Executable))
		et.workDir = filepath.Join(steamAppInstallDir, windowsToNixPath(slc.WorkingDir))
		et.prefix = absPrefixDir
		et.title = slc.Description
		if et.title == "" {
			et.title = appInfoName
		}

		return et, nil
	}

	return nil, errors.New("named steam launch config not found for " + steamAppId)
}
