package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

const exeExt = ".exe"

const (
	relSteamAppsCommonPath        = "Steam/steamapps/common"
	relSteamCompatibilityToolPath = "Steam/compatibilitytools.d"
)

func linuxProtonExecTask(id string, et *execTask) error {

	lwra := nod.Begin(" running %s with Proton, please wait...", et.title)
	defer lwra.Done()

	if et.verbose && len(et.env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(et.env, " "))
	}

	if et.verbose && len(et.protonOptions) > 0 {
		poa := nod.Begin(" proton:")
		poa.EndWithResult(strings.Join(et.protonOptions, " "))
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

	absProtonPath, err := linuxProtonRuntimePath(et, rdx)
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

	for _, option := range et.protonOptions {
		if optionEnv, ok := wine_integration.ProtonOptionsEnv[option]; ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=1", optionEnv))
		}
	}

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
		if err = os.MkdirAll(absPrefixDir, pathways.PermUrwGrwOr); err != nil {
			return err
		}
	}

	return nil
}

func linuxProtonRuntimePath(et *execTask, rdx redux.Readable) (string, error) {

	if et.steamProtonRuntime != "" {

		udhd, err := data.UserDataHomeDir()
		if err != nil {
			return "", err
		}

		absCompatibilityToolPath := filepath.Join(udhd, relSteamCompatibilityToolPath, wine_integration.SteamProtonDirectories[et.steamProtonRuntime])
		if _, err = os.Stat(absCompatibilityToolPath); err == nil {
			return absCompatibilityToolPath, nil
		}

		absSteamAppsCommonPath := filepath.Join(udhd, relSteamAppsCommonPath, wine_integration.SteamProtonDirectories[et.steamProtonRuntime])
		if _, err = os.Stat(absSteamAppsCommonPath); err == nil {
			return absSteamAppsCommonPath, nil
		}

		return "", errors.New("steam proton runtime not found")
	} else {
		return data.ProtonLatestReleasePath(et.protonRuntime, rdx)
	}
}
