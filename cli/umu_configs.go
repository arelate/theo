package cli

import (
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
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

func getLatestUmuConfigsDir() (string, error) {
	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return "", err
	}

	kvGitHubReleases, err := kevlar.New(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return "", err
	}

	latestUmuLauncherRelease, err := github_integration.GetLatestRelease(github_integration.UmuLauncherRepo, kvGitHubReleases)
	if err != nil {
		return "", err
	}

	umuConfigsDir, err := pathways.GetAbsRelDir(data.UmuConfigs)
	if err != nil {
		return "", err
	}

	latestUmuConfigsDir := filepath.Join(umuConfigsDir, busan.Sanitize(latestUmuLauncherRelease.TagName))

	return latestUmuConfigsDir, nil

}

func getAbsUmuConfigFilename(id, exePath string) (string, error) {

	latestUmuConfigsDir, err := getLatestUmuConfigsDir()
	if err != nil {
		return "", err
	}

	_, exeFilename := filepath.Split(exePath)

	umuConfigPath := filepath.Join(latestUmuConfigsDir, id+"-"+busan.Sanitize(exeFilename)+".toml")

	return umuConfigPath, nil
}

func createUmuConfig(cfg *UmuConfig, force bool) (string, error) {

	umuConfigPath, err := getAbsUmuConfigFilename(cfg.GogId, cfg.ExePath)
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
		for _, a := range cfg.Args {
			quotedArgs = append(quotedArgs, "\""+a+"\"")
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

func removeAllUmuConfigs() error {

	rauca := nod.NewProgress("removing all umu-configs...")
	defer rauca.Done()

	latestUmuConfigsDir, err := getLatestUmuConfigsDir()
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

	if err = removeDirIfEmpty(latestUmuConfigsDir); err != nil {
		return err
	}

	return nil
}
