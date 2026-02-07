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

	return SteamRun(id, operatingSystem, et)
}

func SteamRun(id string, operatingSystem vangogh_integration.OperatingSystem, et *execTask) error {

	sra := nod.Begin("running %s for %s...", id, operatingSystem)
	defer sra.Done()

	steamAppInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)
	kvSteamAppInfo, err := kevlar.New(steamAppInfoDir, steam_vdf.Ext)
	if err != nil {
		return err
	}

	appInfoRc, err := kvSteamAppInfo.Get(id)
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
	appInstallDir := filepath.Join(steamAppsDir, operatingSystem.String(), appInfo.Config.InstallDir)

	prefixesDir := data.Pwd.AbsRelDirPath(data.Prefixes, data.Wine)
	et.prefix = filepath.Join(prefixesDir, appInfo.Common.Name)

	// TODO: better detect default launch task
	for _, slc := range appInfo.Config.Launch {
		osList := vangogh_integration.ParseManyOperatingSystems(strings.Split(slc.Config.OsList, ","))
		if slices.Contains(osList, operatingSystem) {

			exe := slc.Executable

			switch operatingSystem {
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

			return osExec(id, operatingSystem, et)
		}
	}

	return nil
}

func windowsToNixPath(wp string) string {
	return strings.Replace(wp, "\\", "/", -1)
}
