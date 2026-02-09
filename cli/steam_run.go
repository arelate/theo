package cli

import (
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arelate/southern_light/steam_appinfo"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
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

	sra := nod.Begin("running %s for %s...", steamAppId, ii.OperatingSystem)
	defer sra.Done()

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

	steamAppsDir := data.Pwd.AbsDirPath(data.SteamApps)
	appInstallDir := filepath.Join(steamAppsDir, ii.OperatingSystem.String(), appInfo.Config.InstallDir)

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

			et.exe = filepath.Join(appInstallDir, exe)
			et.workDir = filepath.Join(appInstallDir, windowsToNixPath(slc.WorkingDir))
			et.name = appInfo.Common.Name
			et.args = append(et.args, strings.Split(slc.Arguments, " ")...)

			return osExec(steamAppId, ii.OperatingSystem, et)
		}
	}

	return nil
}

func windowsToNixPath(wp string) string {
	return strings.Replace(wp, "\\", "/", -1)
}
