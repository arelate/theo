package cli

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arelate/southern_light/steam_appinfo"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const steamAppIdTxt = "steam_appid.txt"

func SteamInstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	createSteamAppId := q.Has("create-steam-appid")

	verbose := q.Has("verbose")
	force := q.Has("force")

	return SteamInstall(id, operatingSystem, createSteamAppId, verbose, force)
}

func SteamInstall(id string, operatingSystem vangogh_integration.OperatingSystem, createSteamAppId, verbose, force bool) error {

	sia := nod.Begin("installing Steam %s for %s...", id, operatingSystem)
	defer sia.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.SteamProperties()...)
	if err != nil {
		return err
	}

	steamAppInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)
	kvSteamAppInfo, err := kevlar.New(steamAppInfoDir, steam_vdf.Ext)
	if err != nil {
		return err
	}

	var username string
	if un, ok := rdx.GetLastVal(data.SteamUsernameProperty, data.SteamUsernameProperty); ok && un != "" {
		username = un
	}

	if err = steamCmdAppInfo(id, username, kvSteamAppInfo, force); err != nil {
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

	operatingSystems := vangogh_integration.ParseManyOperatingSystems(strings.Split(appInfo.Common.OsList, ","))
	if len(operatingSystems) > 0 && !slices.Contains(operatingSystems, operatingSystem) {
		return errors.New(appInfo.Common.Name + " is not available for " + operatingSystem.String())
	} else if len(operatingSystems) == 0 && operatingSystem == vangogh_integration.Windows {
		// do nothing, try to install the default Windows version
	} else if len(operatingSystems) == 0 {
		return errors.New(appInfo.Common.Name + " has no operating systems listed")
	}

	if operatingSystem == vangogh_integration.Windows && data.CurrentOs() != vangogh_integration.Windows {
		if err = steamPrefixInit(appInfo.Common.Name, verbose); err != nil {
			return err
		}
	}

	if err = steamCmdAppUpdate(id, appInfo.Common.Name, username, operatingSystem, appInfo.Config.InstallDir); err != nil {
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
	absInstallDir := filepath.Join(steamAppsDir, operatingSystem.String(), appInfo.Config.InstallDir)

	sgo := &steamGridOptions{
		useSteamAssets: true,
		steamRun:       true,
		name:           appInfo.Common.Name,
		installDir:     absInstallDir,
		logoPosition:   nil,
	}

	if err = SteamShortcut([]string{id}, nil, false, ii, sgo); err != nil {
		return err
	}

	if createSteamAppId {
		// https://partner.steamgames.com/doc/sdk/api
		absSteamAppIdTxtPath := filepath.Join(absInstallDir, steamAppIdTxt)
		if _, err = os.Stat(absSteamAppIdTxtPath); err != nil {
			var sait *os.File
			sait, err = os.Create(absSteamAppIdTxtPath)
			if err != nil {
				return err
			}
			defer sait.Close()

			if _, err = io.WriteString(sait, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func steamCmdAppInfo(id string, username string, kvSteamAppInfo kevlar.KeyValues, force bool) error {

	scaia := nod.Begin(" getting Steam appinfo for %s...", id)
	defer scaia.Done()

	if kvSteamAppInfo.Has(id) && !force {
		scaia.EndWithResult("already exist")
		return nil
	}

	appInfoPrintCmd, err := steamCmdCommand(data.CurrentOs(),
		"+@ShutdownOnFailedCommand", "1",
		"+@NoPromptForPassword", "1",
		"+login", username,
		"+app_info_print", id,
		"+quit")
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
		"+@ShutdownOnFailedCommand", "1",
		"+@NoPromptForPassword", "1",
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
