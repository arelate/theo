package cli

import (
	"errors"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/steam_integration"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func RunHandler(u *url.URL) error {

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

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
		force:           q.Has("force"),
	}

	et := &execTask{
		workDir:         q.Get("work-dir"),
		verbose:         q.Has("verbose"),
		task:            q.Get("task"),
		defaultLauncher: q.Has("default-launcher"),
		noFix:           q.Has("no-fix"),
	}

	if q.Has("env") {
		et.env = strings.Split(q.Get("env"), ",")
	}

	if q.Has("arg") {
		et.args = strings.Split(q.Get("arg"), ",")
	}

	if q.Has("proton-runtime") {
		protonRuntime := q.Get("proton-runtime")
		switch protonRuntime {
		case "umu-proton":
			et.protonRuntime = wine_integration.UmuProton
		case "proton-ge":
			et.protonRuntime = wine_integration.ProtonGe
		}
	}

	if et.protonRuntime == "" {
		et.protonRuntime = wine_integration.ProtonGe
	}

	et.steamProtonRuntime = q.Get("steam-proton-runtime")

	if q.Has("proton-options") {
		et.protonOptions = strings.Split(q.Get("proton-options"), ",")
	}

	return Run(id, ii, et)
}

func Run(id string, request *InstallInfo, et *execTask) error {

	playSessionStart := time.Now()

	ra := nod.NewProgress("running product %s...", id)
	defer ra.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	ii, err := matchInstalledInfo(id, request, rdx)
	if err != nil {
		return err
	}

	vangogh_integration.PrintParams([]string{id},
		[]vangogh_integration.OperatingSystem{ii.OperatingSystem},
		[]string{ii.LangCode},
		ii.DownloadTypes,
		true)

	if err = setLastRunDate(rdx, id); err != nil {
		return err
	}

	originData, err := originGetData(id, ii, rdx, false)
	if err != nil {
		return err
	}

	switch ii.Origin {
	case data.VangoghOrigin:
		if err = checkProductType(id, rdx, ii.force); err != nil {
			return err
		}
		if err = vangoghRun(id, ii, rdx, et); err != nil {
			return err
		}
	case data.SteamOrigin:
		if err = steamRun(id, ii, originData, rdx, et); err != nil {
			return err
		}
	case data.EpicGamesOrigin:
		if err = egsRun(id, ii, originData, rdx, et); err != nil {
			return err
		}
	default:
		return ii.Origin.ErrUnsupportedOrigin()
	}

	playSessionDuration := time.Since(playSessionStart)

	if err = recordPlaytime(rdx, id, playSessionDuration); err != nil {
		return err
	}

	return updateTotalPlaytime(rdx, id)
}

func checkProductType(id string, rdx redux.Writeable, force bool) error {

	productDetails, err := vangoghGetProductDetails(id, rdx, force)
	if err != nil {
		return err
	}

	switch productDetails.ProductType {
	case vangogh_integration.GameProductType:
		// do nothing, proceed normally
		return nil
	case vangogh_integration.PackProductType:
		return errors.New("cannot run a PACK product, please run included game(s): " +
			strings.Join(productDetails.IncludesGames, ","))
	case vangogh_integration.DlcProductType:
		return errors.New("cannot run a DLC product, please run required game(s): " +
			strings.Join(productDetails.RequiresGames, ","))
	default:
		return errors.New("unsupported product type: " + productDetails.ProductType)
	}
}

func setLastRunDate(rdx redux.Writeable, id string) error {

	if err := rdx.MustHave(data.LastRunDateProperty); err != nil {
		return err
	}

	fmtUtcNow := time.Now().UTC().Format(time.RFC3339)
	return rdx.ReplaceValues(data.LastRunDateProperty, id, fmtUtcNow)
}

func recordPlaytime(rdx redux.Writeable, id string, dur time.Duration) error {

	if err := rdx.MustHave(data.PlaytimeMinutesProperty); err != nil {
		return err
	}

	// this will lose some seconds precision
	fmtDur := strconv.FormatInt(int64(dur.Minutes()), 10)

	return rdx.AddValues(data.PlaytimeMinutesProperty, id, fmtDur)
}

func updateTotalPlaytime(rdx redux.Writeable, id string) error {
	if err := rdx.MustHave(data.PlaytimeMinutesProperty, data.TotalPlaytimeMinutesProperty); err != nil {
		return err
	}

	var totalPlaytimeMinutes int64
	if tpms, ok := rdx.GetAllValues(data.PlaytimeMinutesProperty, id); ok && len(tpms) > 0 {
		for _, mins := range tpms {
			if mini, err := strconv.ParseInt(mins, 10, 64); err == nil {
				totalPlaytimeMinutes += mini
			} else {
				return err
			}
		}
	}

	if totalPlaytimeMinutes > 0 {
		return rdx.ReplaceValues(data.TotalPlaytimeMinutesProperty, id, strconv.FormatInt(totalPlaytimeMinutes, 10))
	} else {
		return nil
	}
}

func osConfirmRunnability(operatingSystem vangogh_integration.OperatingSystem) error {
	if operatingSystem == vangogh_integration.MacOS && data.CurrentOs() != vangogh_integration.MacOS {
		return errors.New("running macOS versions is only supported on macOS")
	}
	if operatingSystem == vangogh_integration.Linux && data.CurrentOs() != vangogh_integration.Linux {
		return errors.New("running Linux versions is only supported on Linux")
	}
	return nil
}

func vangoghRun(id string, ii *InstallInfo, rdx redux.Readable, et *execTask) error {

	var err error
	if err = osConfirmRunnability(ii.OperatingSystem); err != nil {
		return err
	}

	if ii.OperatingSystem == vangogh_integration.Windows && data.CurrentOs() != vangogh_integration.Windows {

		var absPrefixDir string
		if absPrefixDir, err = data.AbsPrefixDir(id, ii.Origin, rdx); err == nil {
			et.prefix = absPrefixDir
		} else {
			return err
		}

		if et.exe != "" {
			return osExec(id, ii.OperatingSystem, et)
		}
	}

	var absGogGameInfoPath string
	switch et.defaultLauncher {
	case false:
		absGogGameInfoPath, err = osFindGogGameInfo(id, ii, rdx)
		if err != nil {
			return err
		}
	case true:
		// do nothing
	}

	switch absGogGameInfoPath {
	case "":
		var absDefaultLauncherPath string
		if absDefaultLauncherPath, err = osFindDefaultLauncher(id, ii, rdx); err != nil {
			return err
		}
		if et, err = osExecTaskDefaultLauncher(absDefaultLauncherPath, ii.OperatingSystem, et); err != nil {
			return err
		}
	default:
		if et, err = osExecTaskGogGameInfo(absGogGameInfoPath, ii.OperatingSystem, et); err != nil {
			return err
		}
	}

	if err = osApplyLaunchOptions(id, ii, et, rdx); err != nil {
		return err
	}

	return osExec(id, ii.OperatingSystem, et)
}

func osFindGogGameInfo(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {

	var gogGameInfoPath string
	var err error

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		gogGameInfoPath, err = macOsFindGogGameInfo(id, ii, rdx)
	case vangogh_integration.Linux:
		gogGameInfoPath, err = linuxFindGogGameInfo(id, ii, rdx)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			gogGameInfoPath, err = prefixFindGogGameInfo(id, ii, rdx)
		case vangogh_integration.Windows:
			gogGameInfoPath, err = windowsFindGogGameInfo(id, ii, rdx)
		default:
			return "", currentOs.ErrUnsupported()
		}
	default:
		return "", ii.OperatingSystem.ErrUnsupported()
	}

	if err != nil {
		return "", err
	}

	return gogGameInfoPath, nil
}

func osExecTaskGogGameInfo(absGogGameInfoPath string, operatingSystem vangogh_integration.OperatingSystem, et *execTask) (*execTask, error) {

	_, gogGameInfoFilename := filepath.Split(absGogGameInfoPath)

	eggia := nod.Begin(" running %s...", gogGameInfoFilename)
	defer eggia.Done()

	gogGameInfo, err := gog_integration.GetGogGameInfo(absGogGameInfoPath)
	if err != nil {
		return nil, err
	}

	switch operatingSystem {
	case vangogh_integration.MacOS:
		return macOsExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
	case vangogh_integration.Linux:
		return linuxExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			return macOsExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
		case vangogh_integration.Linux:
			return linuxExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
		case vangogh_integration.Windows:
			return windowsExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
		default:
			return nil, currentOs.ErrUnsupported()
		}
	default:
		return nil, operatingSystem.ErrUnsupported()
	}
}

func osFindDefaultLauncher(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {

	var defaultLauncherPath string
	var err error

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		defaultLauncherPath, err = macOsFindBundleApp(id, ii, rdx)
	case vangogh_integration.Linux:
		defaultLauncherPath, err = linuxFindStartSh(id, ii, rdx)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			defaultLauncherPath, err = prefixFindGogGamesLnk(id, ii, rdx)
		case vangogh_integration.Windows:
			defaultLauncherPath, err = windowsFindGogGamesLnk(id, ii, rdx)
		default:
			return "", currentOs.ErrUnsupported()
		}
	default:
		return "", ii.OperatingSystem.ErrUnsupported()
	}

	if err != nil {
		return "", err
	}

	return defaultLauncherPath, nil
}

func osExecTaskDefaultLauncher(absDefaultLauncherPath string, operatingSystem vangogh_integration.OperatingSystem, et *execTask) (*execTask, error) {

	_, defaultLauncherFilename := filepath.Split(absDefaultLauncherPath)

	et.title = defaultLauncherFilename

	eggia := nod.Begin(" running %s...", defaultLauncherFilename)
	defer eggia.Done()

	switch operatingSystem {
	case vangogh_integration.MacOS:
		return macOsExecTaskBundleApp(absDefaultLauncherPath, et)
	case vangogh_integration.Linux:
		return linuxExecTaskStartSh(absDefaultLauncherPath, et)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			et.exe = absDefaultLauncherPath
		case vangogh_integration.Windows:
			return windowsExecTaskLnk(absDefaultLauncherPath, et)
		default:
			return nil, currentOs.ErrUnsupported()
		}
	default:
		return nil, operatingSystem.ErrUnsupported()
	}

	return et, nil
}

func osExec(id string, operatingSystem vangogh_integration.OperatingSystem, et *execTask) error {

	switch operatingSystem {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		return nixRunExecTask(et)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			return macOsWineExecTask(id, et)
		case vangogh_integration.Linux:
			return linuxProtonExecTask(id, et)
		default:
			return currentOs.ErrUnsupported()
		}
	default:
		return operatingSystem.ErrUnsupported()
	}
}

func windowsToNixPath(wp string) string {
	return strings.Replace(wp, "\\", "/", -1)
}

func steamRun(steamAppId string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable, et *execTask) error {

	var err error

	switch et.task {
	case "":
		et, err = steamDefaultTask(steamAppId, originData.AppInfoKv, ii, rdx)
		if err != nil {
			return err
		}
	default:
		et, err = steamNamedTask(steamAppId, et.task, originData.AppInfoKv, ii, rdx)
		if err != nil {
			return err
		}
	}

	if err = osApplyLaunchOptions(steamAppId, ii, et, rdx); err != nil {
		return err
	}

	return osExec(steamAppId, ii.OperatingSystem, et)
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

func egsRun(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable, et *execTask) error {

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return err
	}

	absPrefixDir, err := data.AbsPrefixDir(appName, ii.Origin, rdx)
	if err != nil {
		return err
	}

	launchDir, launchFile := filepath.Split(originData.Manifest.Metadata.LaunchExe)

	et.title = launchFile
	et.prefix = absPrefixDir
	et.exe = filepath.Join(installedPath, originData.Manifest.Metadata.LaunchExe)
	if originData.Manifest.Metadata.LaunchCommand != "" {
		et.args = append(et.args, originData.Manifest.Metadata.LaunchCommand)
	}
	et.workDir = filepath.Join(installedPath, launchDir)

	if err = osApplyLaunchOptions(appName, ii, et, rdx); err != nil {
		return err
	}

	return osExec(appName, ii.OperatingSystem, et)
}

func osApplyLaunchOptions(id string, ii *InstallInfo, et *execTask, rdx redux.Readable) error {

	if err := rdx.MustHave(
		data.LaunchOptionsExeProperty,
		data.LaunchOptionsArgProperty,
		data.LaunchOptionsEnvProperty); err != nil {
		return err
	}

	appOsLangCode := data.AppOsLangCode(id, ii.OperatingSystem, ii.LangCode)

	if exe, ok := rdx.GetLastVal(data.LaunchOptionsExeProperty, appOsLangCode); ok && exe != "" {
		et.exe = exe
	}

	if args, ok := rdx.GetLastVal(data.LaunchOptionsArgProperty, appOsLangCode); ok && len(args) > 0 {
		et.args = append(et.args, args)
	}

	if env, ok := rdx.GetLastVal(data.LaunchOptionsEnvProperty, appOsLangCode); ok && len(env) > 0 {
		et.env = append(et.env, env)
	}

	return nil
}
