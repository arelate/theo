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

func InitWinePrefix(wineBinPath, absPrefixPath string) error {
	return wineCmd(wineBinPath, absPrefixPath, winebootBin, "--init")
}

func UpdateWinePrefix(wineBinPath, absPrefixPath string) error {
	return wineCmd(wineBinPath, absPrefixPath, winebootBin, "--update")
}

func WinePrefixEnv(absPrefixPath string) string {
	return strings.Join([]string{WinePrefixEnvVar, absPrefixPath}, "=")
}

func wineCmd(wineBinPath, absPrefixPath string, args ...string) error {
	cmd := exec.Command(wineBinPath, args...)
	cmd.Env = append(cmd.Env, WinePrefixEnv(absPrefixPath))
	return cmd.Run()
}
