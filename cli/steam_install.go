package cli

import (
	"bufio"
	"bytes"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
)

func SteamInstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	username := q.Get("username")

	verbose := q.Has("verbose")
	force := q.Has("force")

	return SteamInstall(id, username, operatingSystem, verbose, force)
}

func SteamInstall(id, username string, operatingSystem vangogh_integration.OperatingSystem, verbose, force bool) error {

	sia := nod.Begin("installing Steam %s for %s...", id, operatingSystem)
	defer sia.Done()

	steamAppInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)
	kvSteamAppInfo, err := kevlar.New(steamAppInfoDir, ".vdf")
	if err != nil {
		return err
	}

	if err = steamCmdAppInfo(id, kvSteamAppInfo, force); err != nil {
		return err
	}

	// TODO: Replace with proper KeyValues.Get
	// TODO: Parse into proper structure
	appInfoKeyValues, err := steam_vdf.ParseText(filepath.Join(steamAppInfoDir, id+".vdf"))
	if err != nil {
		return err
	}

	var name string
	if nameKv := steam_vdf.GetKevValuesByKey(appInfoKeyValues, "name"); nameKv != nil {
		if nkv := nameKv.Value; nkv != nil {
			name = *nkv
		}
	}

	var operatingSystems []vangogh_integration.OperatingSystem
	if osListKv := steam_vdf.GetKevValuesByKey(appInfoKeyValues, "oslist"); osListKv != nil {
		if oslv := osListKv.Value; oslv != nil {
			if osList := strings.Split(*oslv, ","); len(osList) > 0 {
				operatingSystems = vangogh_integration.ParseManyOperatingSystems(osList)
				if !slices.Contains(operatingSystems, operatingSystem) {
					return errors.New(name + " is not available for " + operatingSystem.String())
				}
			}
		}
	}

	var installDir string
	if installDirKv := steam_vdf.GetKevValuesByKey(appInfoKeyValues, "installdir"); installDirKv != nil {
		if idv := installDirKv.Value; idv != nil {
			installDir = *idv
		}
	}

	if operatingSystem == vangogh_integration.Windows && data.CurrentOs() != vangogh_integration.Windows {
		if err = steamPrefixInit(name, verbose); err != nil {
			return err
		}
	}

	if err = steamCmdAppUpdate(id, name, username, operatingSystem, installDir); err != nil {
		return err
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        defaultLangCode,
		UseSteamAssets:  true,
		verbose:         verbose,
		force:           force,
	}

	steamAppsDir := data.Pwd.AbsDirPath(data.SteamApps)
	absInstallDir := filepath.Join(steamAppsDir, operatingSystem.String(), installDir)

	sgo := &steamGridOptions{
		useSteamAssets: true,
		steamRun:       true,
		name:           name,
		installDir:     absInstallDir,
		logoPosition:   nil,
	}

	if err = SteamShortcut([]string{id}, nil, false, ii, sgo); err != nil {
		return err
	}

	return nil
}

func steamCmdAppInfo(id string, kvSteamAppInfo kevlar.KeyValues, force bool) error {

	scaia := nod.Begin(" getting Steam appinfo for %s...", id)
	defer scaia.Done()

	if kvSteamAppInfo.Has(id) && !force {
		scaia.EndWithResult("already exist")
		return nil
	}

	appInfoPrintCmd, err := steamCmdCommand(data.CurrentOs(), "+app_info_print", id, "+quit")
	if err != nil {
		return err
	}

	stdout := bytes.NewBuffer(nil)

	appInfoPrintCmd.Stdout = stdout
	appInfoPrintCmd.Stderr = stdout

	if err = appInfoPrintCmd.Run(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	sb := new(strings.Builder)
	appinfo := false

	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case "\"" + id + "\"":
			appinfo = true
		case "}":
			sb.WriteString(line)
			appinfo = false
		default:
			// do nothing
		}

		if appinfo {
			sb.WriteString(line)
		}
	}

	if scanner.Err() != nil {
		return err
	}

	if err = kvSteamAppInfo.Set(id, strings.NewReader(sb.String())); err != nil {
		return err
	}

	return nil
}

func steamCmdAppUpdate(id, name string, username string, operatingSystem vangogh_integration.OperatingSystem, installDir string) error {

	scaua := nod.Begin("downloading %s (%s) for %s with SteamCMD, please wait...", name, id, operatingSystem)
	defer scaua.Done()

	steamAppsDir := data.Pwd.AbsDirPath(data.SteamApps)
	absInstallDir := filepath.Join(steamAppsDir, operatingSystem.String(), installDir)

	if _, err := os.Stat(absInstallDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absInstallDir, 0755); err != nil {
			return err
		}
	}

	steamOs := strings.ToLower(operatingSystem.String())

	steamAppUpdateCmd, err := steamCmdCommand(data.CurrentOs(),
		"+@sSteamCmdForcePlatformType", steamOs,
		"+force_install_dir", absInstallDir,
		"+login", username,
		"+app_update", id,
		"+quit")
	if err != nil {
		return err
	}

	return steamAppUpdateCmd.Run()
}
