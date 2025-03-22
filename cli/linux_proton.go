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

func linuxProtonRun(id, langCode string, rdx redux.Readable, env []string, verbose, force bool, exePath string, arg ...string) error {

	_, exeFilename := filepath.Split(exePath)

	lwra := nod.Begin(" running %s with WINE, please wait...", exeFilename)
	defer lwra.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
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

	absUmuConfigPath, err := createUmuConfig(id, absPrefixDir, absProtonPath, exePath, "gog", force, arg...)
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

	latestUmuLauncherRelease, err := github_integration.UmuLauncher.GetLatestRelease(kvGitHubReleases)
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

func createUmuConfig(id, prefix, proton, exePath, store string, force bool, arg ...string) (string, error) {

	umuConfigPath, err := getAbsUmuConfigFilename(id, exePath)
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
	if _, err = io.WriteString(umuConfigFile, "prefix = \""+prefix+"\"\n"); err != nil {
		return "", err
	}
	if _, err = io.WriteString(umuConfigFile, "proton = \""+proton+"\"\n"); err != nil {
		return "", err
	}
	if _, err = io.WriteString(umuConfigFile, "game_id = \""+id+"\"\n"); err != nil {
		return "", err
	}
	if _, err = io.WriteString(umuConfigFile, "exe = \""+exePath+"\"\n"); err != nil {
		return "", err
	}
	if len(arg) > 0 {
		if _, err = io.WriteString(umuConfigFile, "launch_args = ["); err != nil {
			return "", err
		}
		quotedArgs := make([]string, 0, len(arg))
		for _, a := range arg {
			quotedArgs = append(quotedArgs, "\""+a+"\"")
		}
		if _, err = io.WriteString(umuConfigFile, strings.Join(quotedArgs, ", ")); err != nil {
			return "", err
		}
		if _, err = io.WriteString(umuConfigFile, "]\n"); err != nil {
			return "", err
		}

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
