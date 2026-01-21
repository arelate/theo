package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

const (
	umuGogStore = "gog"
)

type UmuConfig struct {
	GogId   string
	Prefix  string
	Proton  string
	ExePath string
	Args    []string
}

func getLatestUmuConfigsDir(rdx redux.Readable) (string, error) {

	runtime := wine_integration.UmuLauncher

	if err := rdx.MustHave(data.WineBinariesVersionsProperty); err != nil {
		return "", err
	}

	var latestUmuLauncherVersion string
	if lulv, ok := rdx.GetLastVal(data.WineBinariesVersionsProperty, runtime); ok {
		latestUmuLauncherVersion = lulv
	}

	if latestUmuLauncherVersion == "" {
		return "", errors.New("umu-launcher version not found, please run setup-wine")
	}

	umuConfigsDir := data.Pwd.AbsRelDirPath(data.UmuConfigs, data.Wine)
	latestUmuConfigsDir := filepath.Join(umuConfigsDir, latestUmuLauncherVersion)

	return latestUmuConfigsDir, nil
}

func getAbsUmuConfigFilename(id, exePath string, rdx redux.Readable) (string, error) {

	latestUmuConfigsDir, err := getLatestUmuConfigsDir(rdx)
	if err != nil {
		return "", err
	}

	_, exeFilename := filepath.Split(exePath)

	umuConfigPath := filepath.Join(latestUmuConfigsDir, id+"-"+pathways.Sanitize(exeFilename)+".toml")

	return umuConfigPath, nil
}

func createUmuConfig(cfg *UmuConfig, rdx redux.Readable) (string, error) {

	umuConfigPath, err := getAbsUmuConfigFilename(cfg.GogId, cfg.ExePath, rdx)
	if err != nil {
		return "", err
	}

	// umu-config should always be recreated to avoid any stale misconfiguration errors, like
	// proton-runtime, prefix, etc.
	//if _, err = os.Stat(umuConfigPath); err == nil && !force {
	//	return umuConfigPath, nil
	//}

	umuConfigDir, _ := filepath.Split(umuConfigPath)
	if _, err = os.Stat(umuConfigDir); os.IsNotExist(err) {
		if err = os.MkdirAll(umuConfigDir, 0755); err != nil {
			return "", err
		}
	}

	umuConfigFile, err := os.Create(umuConfigPath)
	if err != nil {
		return "", err
	}

	defer umuConfigFile.Close()

	escapedArgs := make([]string, 0, len(cfg.Args))
	for _, arg := range cfg.Args {
		//ea := strings.Replace(a, "\"", "\\\"", -1)
		ea := strings.Replace(arg, "\\", "\\\\", -1)
		ea = strings.Replace(ea, "\"", "\\\"", -1)
		escapedArgs = append(escapedArgs, ea)
	}

	if _, err = io.WriteString(umuConfigFile, "[umu]\n"); err != nil {
		return "", err
	}
	if _, err = io.WriteString(umuConfigFile, "prefix = \""+cfg.Prefix+"\"\n"); err != nil {
		return "", err
	}
	if _, err = io.WriteString(umuConfigFile, "proton = \""+cfg.Proton+"\"\n"); err != nil {
		return "", err
	}
	if _, err = io.WriteString(umuConfigFile, "exe = \""+cfg.ExePath+"\"\n"); err != nil {
		return "", err
	}
	if len(cfg.Args) > 0 {
		if _, err = io.WriteString(umuConfigFile, "launch_args = ["); err != nil {
			return "", err
		}
		quotedArgs := make([]string, 0, len(cfg.Args))
		for _, ea := range escapedArgs {
			quotedArgs = append(quotedArgs, "\""+ea+"\"")
		}
		if _, err = io.WriteString(umuConfigFile, strings.Join(quotedArgs, ", ")); err != nil {
			return "", err
		}
		if _, err = io.WriteString(umuConfigFile, "]\n"); err != nil {
			return "", err
		}
	}

	if _, err = io.WriteString(umuConfigFile, "game_id = \""+cfg.GogId+"\"\n"); err != nil {
		return "", err
	}
	if _, err = io.WriteString(umuConfigFile, "store = \""+umuGogStore+"\"\n"); err != nil {
		return "", err
	}

	return umuConfigPath, nil
}
