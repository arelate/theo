package cli

import (
	"errors"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	umuGogStore   = "gog"
	umuSteamStore = "steam"
)

type UmuConfig struct {
	GogId      string
	SteamAppId string
	Prefix     string
	Proton     string
	ExePath    string
	Args       []string
}

func getLatestUmuConfigsDir(rdx redux.Readable) (string, error) {

	runtime := vangogh_integration.UmuLauncher

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

	umuConfigsDir, err := pathways.GetAbsRelDir(data.UmuConfigs)
	if err != nil {
		return "", err
	}

	latestUmuConfigsDir := filepath.Join(umuConfigsDir, latestUmuLauncherVersion)

	fmt.Println(latestUmuConfigsDir)

	if _, err = os.Stat(latestUmuConfigsDir); err == nil {
		return latestUmuConfigsDir, nil
	}

	return "", os.ErrNotExist
}

func getAbsUmuConfigFilename(id, exePath string, rdx redux.Readable) (string, error) {

	latestUmuConfigsDir, err := getLatestUmuConfigsDir(rdx)
	if err != nil {
		return "", err
	}

	_, exeFilename := filepath.Split(exePath)

	umuConfigPath := filepath.Join(latestUmuConfigsDir, id+"-"+busan.Sanitize(exeFilename)+".toml")

	return umuConfigPath, nil
}

func createUmuConfig(cfg *UmuConfig, rdx redux.Readable, force bool) (string, error) {

	umuConfigPath, err := getAbsUmuConfigFilename(cfg.GogId, cfg.ExePath, rdx)
	if err != nil {
		return "", err
	}

	if _, err = os.Stat(umuConfigPath); err == nil && !force {
		return umuConfigPath, nil
	}

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

	var id, store string

	if cfg.SteamAppId != "" {
		id = cfg.SteamAppId
		store = umuSteamStore
	} else {
		id = cfg.GogId
		store = umuGogStore
	}

	if _, err = io.WriteString(umuConfigFile, "game_id = \""+id+"\"\n"); err != nil {
		return "", err
	}
	if _, err = io.WriteString(umuConfigFile, "store = \""+store+"\"\n"); err != nil {
		return "", err
	}

	return umuConfigPath, nil
}

func resetUmuConfigs(rdx redux.Readable) error {

	rauca := nod.NewProgress("resetting umu-configs...")
	defer rauca.Done()

	latestUmuConfigsDir, err := getLatestUmuConfigsDir(rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(latestUmuConfigsDir); os.IsNotExist(err) {
		return nil
	}

	lucd, err := os.Open(latestUmuConfigsDir)
	if err != nil {
		return err
	}

	relFilenames, err := lucd.Readdirnames(-1)
	if err != nil {
		return err
	}

	rauca.TotalInt(len(relFilenames))

	for _, rfn := range relFilenames {
		if strings.HasPrefix(rfn, ".") {
			rauca.Increment()
			continue
		}

		afn := filepath.Join(latestUmuConfigsDir, rfn)
		if err = os.Remove(afn); err != nil {
			return err
		}

		rauca.Increment()
	}

	var empty bool
	if empty, err = osIsDirEmpty(latestUmuConfigsDir); empty && err == nil {
		if err = os.RemoveAll(latestUmuConfigsDir); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}
