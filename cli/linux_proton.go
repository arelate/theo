package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func linuxProtonRun(id, langCode string, rdx redux.Readable, et *execTask, force bool) error {

	_, exeFilename := filepath.Split(et.exe)

	lwra := nod.Begin(" running %s with WINE, please wait...", exeFilename)
	defer lwra.Done()

	if err := rdx.MustHave(
		vangogh_integration.SlugProperty,
		vangogh_integration.SteamAppIdProperty); err != nil {
		return err
	}

	if et.verbose && len(et.env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(et.env, " "))
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
		ExePath: et.exe,
		Args:    et.args,
	}

	if steamAppId, ok := rdx.GetLastVal(vangogh_integration.SteamAppIdProperty, id); ok && steamAppId != "" {
		umuCfg.SteamAppId = steamAppId
	}

	absUmuConfigPath, err := createUmuConfig(umuCfg, force)
	if err != nil {
		return err
	}

	cmd := exec.Command(absUmuRunPath, "--config", absUmuConfigPath)

	if et.workDir != "" {
		cmd.Dir = et.workDir
	}

	cmd.Env = append(os.Environ(), et.env...)

	if et.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

func linuxProtonRunExecTask(gogId, steamAppId string, et *execTask, force bool) error {

	lwra := nod.Begin(" running %s with Proton, please wait...", et.name)
	defer lwra.Done()

	if et.verbose && len(et.env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(et.env, " "))
	}

	absUmuRunPath, err := data.UmuRunLatestReleasePath()
	if err != nil {
		return err
	}

	absProtonPath, err := data.UmuProtonLatestReleasePath()
	if err != nil {
		return err
	}

	umuCfg := &UmuConfig{
		GogId:      gogId,
		SteamAppId: steamAppId,
		Prefix:     et.prefix,
		Proton:     absProtonPath,
		ExePath:    et.exe,
		Args:       et.args,
	}

	absUmuConfigPath, err := createUmuConfig(umuCfg, force)
	if err != nil {
		return err
	}

	cmd := exec.Command(absUmuRunPath, "--config", absUmuConfigPath)

	if et.workDir != "" {
		cmd.Dir = et.workDir
	}

	cmd.Env = append(os.Environ(), et.env...)

	if et.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
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
