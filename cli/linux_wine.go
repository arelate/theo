package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func linuxWineRun(id, langCode string, env []string, verbose, force bool, exePath string, arg ...string) error {

	_, exeFilename := filepath.Split(exePath)

	lwra := nod.Begin(" running %s with WINE, please wait...", exeFilename)
	defer lwra.EndWithResult("done")

	if verbose && len(env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(env, " "))
	}

	absUmuRunPath, err := data.UmuRunLatestReleasePath()
	if err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
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

	umuConfigsDir, err := pathways.GetAbsRelDir(data.UmuConfigs)
	if err != nil {
		return "", err
	}

	_, exeFilename := filepath.Split(exePath)

	umuConfigPath := filepath.Join(umuConfigsDir, id+"-"+busan.Sanitize(exeFilename)+".toml")

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

func linuxStartGogGamesLnk(id, langCode string, env []string, verbose, force bool, arg ...string) error {
	lsggla := nod.Begin(" starting default .lnk in the install folder for %s...", id)
	defer lsggla.EndWithResult("done")

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
	if err != nil {
		return lsggla.EndWithError(err)
	}

	absPrefixDriveCDir := filepath.Join(absPrefixDir, relPrefixDriveCDir)

	matches, err := filepath.Glob(filepath.Join(absPrefixDriveCDir, gogInstallationLnkGlob))
	if err != nil {
		return lsggla.EndWithError(err)
	}

	if len(matches) == 1 {

		firstMatch := matches[0]

		relMatch, err := filepath.Rel(absPrefixDriveCDir, firstMatch)
		if err != nil {
			return lsggla.EndWithError(err)
		}

		lsggla.EndWithResult("found %s", filepath.Join("C:", relMatch))

		return linuxWineRun(id, langCode, env, verbose, force, firstMatch, arg...)
	} else {
		return lsggla.EndWithError(errors.New("cannot locate suitable .lnk in the GOG Games folder"))
	}
}

func linuxInitPrefix(id, langCode string, _ bool) error {
	lipa := nod.Begin(" initializing prefix...")
	defer lipa.EndWithResult("done")

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
	if err != nil {
		return lipa.EndWithError(err)
	}

	return os.MkdirAll(absPrefixDir, 0755)
}
