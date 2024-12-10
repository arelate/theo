package data

import (
	"errors"
	"fmt"
	"github.com/arelate/vangogh_local_data"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	winePfxEnvVar   = "WINEPREFIX"
	RelPfxDriveCDir = "drive_c"
)

const (
	winebootBin = "wineboot"
	regeditBin  = "regedit"
)

const (
	initFlag   = "--init"
	updateFlag = "--update"
)

type WineContext struct {
	BinPath    string
	PrefixPath string
}

func GetWineBinary(os vangogh_local_data.OperatingSystem, releaseSelector *GitHubReleaseSelector) (string, error) {

	wineSource, release, err := GetWineSourceRelease(os, releaseSelector)
	if err != nil {
		return "", err
	}

	if wineSource == nil {
		return "", errors.New("nil wine source selected")
	}

	binDir, err := GetAbsBinariesDir(wineSource.GitHubSource, release)
	if err != nil {
		return "", err
	}

	return filepath.Join(binDir, wineSource.BinaryPath), nil
}

func InitWinePrefix(wcx *WineContext) error {
	env := map[string]string{
		winePfxEnvVar: wcx.PrefixPath,
	}
	return wineCmd(wcx.BinPath, env, winebootBin, initFlag)
}

func UpdateWinePrefix(wcx *WineContext) error {
	env := map[string]string{
		winePfxEnvVar: wcx.PrefixPath,
	}
	return wineCmd(wcx.BinPath, env, winebootBin, updateFlag)
}

func RegeditWinePrefix(wcx *WineContext, absRegPath string) error {
	env := map[string]string{
		winePfxEnvVar: wcx.PrefixPath,
	}
	return wineCmd(wcx.BinPath, env, regeditBin, absRegPath)
}

func wineCmd(absWineBinPath string, env map[string]string, args ...string) error {
	cmd := exec.Command(absWineBinPath, args...)
	for p, v := range env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", p, v))
	}
	return cmd.Run()
}
