package cli

import (
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arelate/southern_light/steam_appinfo"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
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

	verbose := q.Has("verbose")

	return SteamRun(id, operatingSystem, verbose)
}

func SteamRun(id string, operatingSystem vangogh_integration.OperatingSystem, verbose bool) error {

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
	absPrefixDir := filepath.Join(prefixesDir, appInfo.Common.Name)

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

			absExePath := filepath.Join(appInstallDir, exe)

			absWorkingDir := filepath.Join(appInstallDir, windowsToNixPath(slc.WorkingDir))

			et := &execTask{
				name:            appInfo.Common.Name,
				exe:             absExePath,
				prefix:          absPrefixDir,
				workDir:         absWorkingDir,
				args:            strings.Split(slc.Arguments, " "),
				defaultLauncher: false,
				verbose:         verbose,
			}
			return osExec(id, operatingSystem, et)
		}
	}

	return nil
}

func windowsToNixPath(wp string) string {
	return strings.Replace(wp, "\\", "/", -1)
}
