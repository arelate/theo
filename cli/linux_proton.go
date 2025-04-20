package cli

import (
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"os"
	"os/exec"
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

func linuxProtonRun(id, langCode string, rdx redux.Readable, env []string, verbose, force bool, exePath string, arg ...string) error {

	_, exeFilename := filepath.Split(exePath)

	lwra := nod.Begin(" running %s with WINE, please wait...", exeFilename)
	defer lwra.Done()

	if err := rdx.MustHave(
		vangogh_integration.SlugProperty,
		vangogh_integration.SteamAppIdProperty); err != nil {
		return err
	}

	if verbose && len(env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(env, " "))
	}

	absUmuRunPath, err := data.UmuRunLatestReleasePath()
	if err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	absProtonPath, err := data.UmuProtonLatestReleasePath()
	if err != nil {
		return err
	}

	umuCfg := &UmuConfig{
		GogId:   id,
		Prefix:  absPrefixDir,
		Proton:  absProtonPath,
		ExePath: exePath,
		Args:    arg,
	}

	if steamAppId, ok := rdx.GetLastVal(vangogh_integration.SteamAppIdProperty, id); ok && steamAppId != "" {
		umuCfg.SteamAppId = steamAppId
	}

	absUmuConfigPath, err := createUmuConfig(umuCfg, force)
	if err != nil {
		return err
	}

	cmd := exec.Command(absUmuRunPath, "--config", absUmuConfigPath)

	cmd.Env = append(os.Environ(), env...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

func getAbsUmuConfigFilename(id, exePath string) (string, error) {

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

	_, exeFilename := filepath.Split(exePath)

	umuConfigPath := filepath.Join(umuConfigsDir,
		busan.Sanitize(latestUmuLauncherRelease.TagName),
		id+"-"+busan.Sanitize(exeFilename)+".toml")

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

func linuxInitPrefix(id, langCode string, rdx redux.Readable, _ bool) error {
	lipa := nod.Begin(" initializing prefix...")
	defer lipa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	return os.MkdirAll(absPrefixDir, 0755)
}
