package data

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const winePrefixEnvVar = "WINEPREFIX"

const RelPrefixDriveCDir = "drive_c"

const gogLnkGlob = "GOG Games/*/*.lnk"

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

func GetWineBinary(wineRepo string) (string, error) {

	wineSource, release, err := GetWineSourceLatestRelease(wineRepo)
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
		winePrefixEnvVar: wcx.PrefixPath,
	}
	return wineCmd(wcx.BinPath, env, winebootBin, initFlag)
}

func UpdateWinePrefix(wcx *WineContext) error {
	env := map[string]string{
		winePrefixEnvVar: wcx.PrefixPath,
	}
	return wineCmd(wcx.BinPath, env, winebootBin, updateFlag)
}

func RegeditWinePrefix(wcx *WineContext, absRegPath string) error {
	env := map[string]string{
		winePrefixEnvVar: wcx.PrefixPath,
	}
	return wineCmd(wcx.BinPath, env, regeditBin, absRegPath)
}

func RunWineInnoExtractInstaller(wcx *WineContext, absInstallerPath, slug string) error {
	env := map[string]string{
		winePrefixEnvVar: wcx.PrefixPath,
	}

	return wineCmd(wcx.BinPath, env, absInstallerPath, "/VERYSILENT", "/NORESTART", "/CLOSEAPPLICATIONS")
}

func RunWineDefaultGogLnk(wcx *WineContext) error {
	env := map[string]string{
		winePrefixEnvVar: wcx.PrefixPath,
	}

	matches, err := filepath.Glob(filepath.Join(wcx.PrefixPath, RelPrefixDriveCDir, gogLnkGlob))
	if err != nil {
		return err
	}

	if len(matches) == 1 {
		return wineCmd(wcx.BinPath, env, matches[0])
	} else {
		return errors.New("cannot locate suitable .lnk in the default GOG install folder")
	}
}

func RunWineExePath(wcx *WineContext, exePath string) error {
	env := map[string]string{
		winePrefixEnvVar: wcx.PrefixPath,
	}

	return wineCmd(wcx.BinPath, env, exePath)
}

func wineCmd(absWineBinPath string, env map[string]string, args ...string) error {

	cmd := exec.Command(absWineBinPath, args...)

	dir, _ := filepath.Split(absWineBinPath)

	cmd.Dir = dir

	for p, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", p, v))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}
