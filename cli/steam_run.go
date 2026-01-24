package cli

import (
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
)

func SteamRunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	return SteamRun(id, operatingSystem)
}

func SteamRun(id string, operatingSystem vangogh_integration.OperatingSystem) error {

	sra := nod.Begin("running %s for %s...", id, operatingSystem)
	defer sra.Done()

	steamAppInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)

	appInfoKeyValues, err := steam_vdf.ParseText(filepath.Join(steamAppInfoDir, id+".vdf"))
	if err != nil {
		return err
	}

	var slcs []*steamLaunchConfig
	if launchKv := steam_vdf.GetKevValuesByKey(appInfoKeyValues, "launch"); launchKv != nil {
		slcs = parseLaunchOptions(launchKv)
	}

	var installDir string
	if installDirKv := steam_vdf.GetKevValuesByKey(appInfoKeyValues, "installdir"); installDirKv != nil {
		if idv := installDirKv.Value; idv != nil {
			installDir = *idv
		}
	}

	var name string
	if nameKv := steam_vdf.GetKevValuesByKey(appInfoKeyValues, "name"); nameKv != nil {
		if nkv := nameKv.Value; nkv != nil {
			name = *nkv
		}
	}

	steamAppsDir := data.Pwd.AbsDirPath(data.SteamApps)
	appInstallDir := filepath.Join(steamAppsDir, operatingSystem.String(), installDir)

	for _, slc := range slcs {
		if slices.Contains(slc.osList, operatingSystem) {

			exe := slc.executable

			switch operatingSystem {
			case vangogh_integration.MacOS:
				fallthrough
			case vangogh_integration.Linux:
				exe = strings.Replace(exe, "\\", "/", -1)
			default:
				// do nothin
			}

			et := &execTask{
				name:            name,
				exe:             filepath.Join(appInstallDir, exe),
				workDir:         slc.workingDir,
				args:            strings.Split(slc.arguments, " "),
				defaultLauncher: false,
				verbose:         false,
			}
			return osExec(id, operatingSystem, et)
		}
	}

	return nil
}

type steamLaunchConfig struct {
	executable  string
	arguments   string
	workingDir  string
	typeStr     string
	osList      []vangogh_integration.OperatingSystem
	osArch      string
	description string
}

func parseLaunchOptions(launchKv *steam_vdf.KeyValues) []*steamLaunchConfig {
	slcs := make([]*steamLaunchConfig, 0, len(launchKv.Values))
	for _, launchValueKey := range launchKv.Values {
		slc := &steamLaunchConfig{}
		for _, launchValue := range launchValueKey.Values {
			value := launchValue.Value
			switch launchValue.Key {
			case "executable":
				slc.executable = *value
			case "arguments":
				slc.arguments = *value
			case "workingdir":
				slc.workingDir = *value
			case "type":
				slc.typeStr = *value
			case "config":
				for _, configValue := range launchValue.Values {
					cv := configValue.Value
					if cv == nil {
						continue
					}
					switch configValue.Key {
					case "oslist":
						slc.osList = vangogh_integration.ParseManyOperatingSystems(strings.Split(*cv, ","))
					case "osarch":
						slc.osArch = *cv
					}
				}
			//
			case "description":
				slc.description = *value

			}
		}
		slcs = append(slcs, slc)
	}
	return slcs
}
