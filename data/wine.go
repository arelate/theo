package data

import (
	"errors"
	"github.com/arelate/vangogh_local_data"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	WinePrefixEnvVar = "WINEPREFIX"
)

const (
	winebootBin = "wineboot"
)

const (
	initFlag   = "--init"
	updateFlag = "--update"
	forceFlag  = "--force"
)

func GetWineBinary(os vangogh_local_data.OperatingSystem, releaseSelector *GitHubReleaseSelector) (string, error) {

	wineSource, release, err := GetWineSourceRelease(os, releaseSelector)
	if err != nil {
		return "", err
	}

	if wineSource == nil {
		return "", errors.New("nil wine source selected")
	}

	binDir, err := GetAbsBinariesDir(&wineSource.GitHubSource, release)
	if err != nil {
		return "", err
	}

	return filepath.Join(binDir, wineSource.BinaryPath), nil
}

func InitWinePrefix(absWineBinPath, absPrefixPath string) error {
	return wineCmd(absWineBinPath, absPrefixPath, winebootBin, initFlag)
}

func UpdateWinePrefix(absWineBinPath, absPrefixPath string) error {
	return wineCmd(absWineBinPath, absPrefixPath, winebootBin, updateFlag)
}

func ForceExitWinePrefix(absWineBinPath, absPrefixPath string) error {
	return wineCmd(absWineBinPath, absPrefixPath, winebootBin, forceFlag)
}

func WinePrefixEnv(absPrefixPath string) string {
	return strings.Join([]string{WinePrefixEnvVar, absPrefixPath}, "=")
}

func wineCmd(absWineBinPath, absPrefixPath string, args ...string) error {
	cmd := exec.Command(absWineBinPath, args...)
	cmd.Env = append(cmd.Env, WinePrefixEnv(absPrefixPath))
	return cmd.Run()
}
