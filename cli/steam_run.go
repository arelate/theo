package cli

import (
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/arelate/southern_light/steam_appinfo"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func SteamRunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
	}

	et := &execTask{
		verbose: q.Has("verbose"),
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

	return SteamRun(id, ii, et)
}

func SteamRun(steamAppId string, ii *InstallInfo, et *execTask) error {

	playSessionStart := time.Now()

	sra := nod.Begin("running %s for %s...", steamAppId, ii.OperatingSystem)
	defer sra.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	if err = resolveInstallInfo(steamAppId, ii, nil, rdx, installedOperatingSystem); err != nil {
		return err
	}

	printInstallInfoParams(ii, true, steamAppId)

	if err = setLastRunDate(rdx, steamAppId); err != nil {
		return err
	}

	steamAppInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)
	kvSteamAppInfo, err := kevlar.New(steamAppInfoDir, steam_vdf.Ext)
	if err != nil {
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

	steamAppInstallDir, err := data.AbsSteamAppInstallDir(steamAppId, ii.OperatingSystem, rdx)
	if err != nil {
		return err
	}

	absSteamPrefixDir, err := data.AbsSteamPrefixDir(steamAppId)
	if err != nil {
		return err
	}

	et.prefix = absSteamPrefixDir

	// TODO: better detect default launch task
	for _, slc := range appInfo.Config.Launch {
		osList := vangogh_integration.ParseManyOperatingSystems(strings.Split(slc.Config.OsList, ","))
		if slices.Contains(osList, ii.OperatingSystem) {

			exe := slc.Executable

			switch ii.OperatingSystem {
			case vangogh_integration.MacOS:
				fallthrough
			case vangogh_integration.Linux:
				exe = windowsToNixPath(exe)
			case vangogh_integration.Windows:
				switch data.CurrentOs() {
				case vangogh_integration.MacOS:
					fallthrough
				case vangogh_integration.Linux:
					exe = windowsToNixPath(exe)
				default:
					return data.CurrentOs().ErrUnsupported()
				}
			default:
				return data.CurrentOs().ErrUnsupported()
			}

			et.exe = filepath.Join(steamAppInstallDir, exe)
			et.workDir = filepath.Join(steamAppInstallDir, windowsToNixPath(slc.WorkingDir))
			et.name = appInfo.Common.Name
			et.args = append(et.args, strings.Split(slc.Arguments, " ")...)

			if err = osExec(steamAppId, ii.OperatingSystem, et); err != nil {
				return err
			}
		}
	}

	playSessionDuration := time.Since(playSessionStart)

	if err = recordPlaytime(rdx, steamAppId, playSessionDuration); err != nil {
		return err
	}

	return updateTotalPlaytime(rdx, steamAppId)
}

func windowsToNixPath(wp string) string {
	return strings.Replace(wp, "\\", "/", -1)
}
