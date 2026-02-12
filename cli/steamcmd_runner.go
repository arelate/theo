package cli

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
)

const (
	steamCmdLoginCommand           = "+login"
	steamCmdForceInstallDirCommand = "+force_install_dir"
	steamCmdAppUpdateCommand       = "+app_update"
	steamCmdAppInfoPrintCommand    = "+app_info_print"
	steamCmdAppUninstallCommand    = "+app_uninstall"
	steamCmdLogoutCommand          = "+logout"
	steamCmdQuitCommand            = "+quit"
)

const (
	steamCmdShutdownOnFailedCommandVariable = "+@ShutdownOnFailedCommand"
	steamCmdNoPromptForPasswordVariable     = "+@NoPromptForPassword"
	steamCmdForcePlatformTypeVariable       = "+@sSteamCmdForcePlatformType"
)

const (
	steamCmdTrueValue      = "1"
	steamCmdFalseValue     = "0"
	steamCmdAnonymousValue = "anonymous"
)

func steamCmdRunner(commands ...string) (*exec.Cmd, error) {

	absSteamCmdBinaryPath, err := data.AbsSteamCmdBinPath(data.CurrentOs())
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(absSteamCmdBinaryPath, commands...)

	// always enable i/o to be able to input Steam password for a given username
	// and see Steam Guard prompt
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, nil
}

func steamCmdLogin(username string) error {

	cmd, err := steamCmdRunner(steamCmdLoginCommand, username, steamCmdQuitCommand)
	if err != nil {
		return err
	}

	return cmd.Run()
}

func steamCmdLogout() error {

	cmd, err := steamCmdRunner(steamCmdLogoutCommand, steamCmdQuitCommand)
	if err != nil {
		return err
	}

	return cmd.Run()
}

func steamCmdAppInfoPrint(id string, username string) (string, error) {

	appInfoPrintCmd, err := steamCmdRunner(
		steamCmdShutdownOnFailedCommandVariable, steamCmdTrueValue,
		steamCmdNoPromptForPasswordVariable, steamCmdTrueValue,
		steamCmdLoginCommand, steamCmdAnonymousValue, // app_info_print works with anonymous login
		steamCmdAppInfoPrintCommand, id,
		steamCmdQuitCommand)
	if err != nil {
		return "", err
	}

	stdout := bytes.NewBuffer(nil)

	appInfoPrintCmd.Stdout = stdout
	appInfoPrintCmd.Stderr = stdout

	if err = appInfoPrintCmd.Run(); err != nil {
		return "", err
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
		return "", err
	}

	return sb.String(), nil
}

func steamCmdAppUpdate(id string, operatingSystem vangogh_integration.OperatingSystem, absInstallDir, username string) error {

	steamAppUpdateCmd, err := steamCmdRunner(
		steamCmdShutdownOnFailedCommandVariable, steamCmdTrueValue,
		steamCmdNoPromptForPasswordVariable, steamCmdTrueValue,
		steamCmdForcePlatformTypeVariable, strings.ToLower(operatingSystem.String()),
		steamCmdForceInstallDirCommand, absInstallDir,
		steamCmdLoginCommand, username,
		steamCmdAppUpdateCommand, id,
		steamCmdQuitCommand)
	if err != nil {
		return err
	}

	return steamAppUpdateCmd.Run()
}

func steamCmdAppUninstall(id string, operatingSystem vangogh_integration.OperatingSystem, absInstallDir string) error {

	steamAppUninstallCmd, err := steamCmdRunner(
		steamCmdShutdownOnFailedCommandVariable, steamCmdTrueValue,
		steamCmdNoPromptForPasswordVariable, steamCmdTrueValue,
		steamCmdForcePlatformTypeVariable, strings.ToLower(operatingSystem.String()),
		steamCmdForceInstallDirCommand, absInstallDir,
		steamCmdLoginCommand, steamCmdAnonymousValue,
		steamCmdAppUninstallCommand, id,
		steamCmdQuitCommand)
	if err != nil {
		return err
	}

	return steamAppUninstallCmd.Run()

}
