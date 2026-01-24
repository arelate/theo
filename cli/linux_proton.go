package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	ProtonEnableWayland = "enable-wayland"
	ProtonPreferSdl     = "prefer-sdl"
	ProtonDisableHidRaw = "disable-hidraw"
	ProtonEnableHdr     = "enable-hdr"
	ProtonNoSteamInput  = "no-steaminput"
)

var protonOptionsEnv = map[string]string{
	ProtonEnableWayland: "PROTON_ENABLE_WAYLAND",
	ProtonPreferSdl:     "PROTON_PREFER_SDL",
	ProtonDisableHidRaw: "PROTON_DISABLE_HIDRAW",
	ProtonEnableHdr:     "PROTON_ENABLE_HDR",
	ProtonNoSteamInput:  "PROTON_NO_STEAMINPUT",
}

func linuxProtonRun(id string, et *execTask) error {

	_, exeFilename := filepath.Split(et.exe)

	lwra := nod.Begin(" running %s with WINE, please wait...", exeFilename)
	defer lwra.Done()

	reduxDir := data.Pwd.AbsRelDirPath(data.Redux, data.Metadata)
	rdx, err := redux.NewReader(reduxDir,
		data.WineBinariesVersionsProperty,
		vangogh_integration.SlugProperty,
		vangogh_integration.SteamAppIdProperty)
	if err != nil {
		return err
	}

	if et.verbose && len(et.env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(et.env, " "))
	}

	absUmuRunPath, err := data.UmuRunLatestReleasePath(rdx)
	if err != nil {
		return err
	}

	absPrefixDir, err := data.AbsPrefixDir(id, rdx)
	if err != nil {
		return err
	}

	absProtonPath, err := data.ProtonLatestReleasePath(et.protonRuntime, rdx)
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

	absUmuConfigPath, err := createUmuConfig(umuCfg, rdx)
	if err != nil {
		return err
	}

	cmd := exec.Command(absUmuRunPath, "--config", absUmuConfigPath)

	if et.workDir != "" {
		cmd.Dir = et.workDir
	}

	cmd.Env = append(os.Environ(), et.env...)

	for _, po := range et.protonOptions {
		if poEnv, ok := protonOptionsEnv[po]; ok {
			cmd.Env = append(cmd.Env, poEnv+"=1")
		}
	}

	if et.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

func linuxProtonRunExecTask(id string, et *execTask) error {

	lwra := nod.Begin(" running %s with Proton, please wait...", et.name)
	defer lwra.Done()

	if et.verbose && len(et.env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(et.env, " "))
	}

	reduxDir := data.Pwd.AbsRelDirPath(data.Redux, data.Metadata)
	rdx, err := redux.NewReader(reduxDir, data.WineBinariesVersionsProperty)
	if err != nil {
		return err
	}

	absUmuRunPath, err := data.UmuRunLatestReleasePath(rdx)
	if err != nil {
		return err
	}

	absProtonPath, err := data.ProtonLatestReleasePath(et.protonRuntime, rdx)
	if err != nil {
		return err
	}

	umuCfg := &UmuConfig{
		GogId:   id,
		Prefix:  et.prefix,
		Proton:  absProtonPath,
		ExePath: et.exe,
		Args:    et.args,
	}

	absUmuConfigPath, err := createUmuConfig(umuCfg, rdx)
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

func linuxInitPrefix(absPrefixDir string, _ bool) error {
	lipa := nod.Begin(" initializing prefix...")
	defer lipa.Done()

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absPrefixDir, 0755); err != nil {
			return err
		}
	}

	return nil
}
