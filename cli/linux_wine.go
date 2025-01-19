package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func linuxWineRun(id, langCode string, env []string, verbose bool, arg ...string) error {
	var cmdArg string
	for _, a := range arg {
		if strings.HasPrefix(a, "-") {
			continue
		}
		_, cmdArg = filepath.Split(a)
		break
	}
	if cmdArg == "" {
		cmdArg = "command"
	}

	lwra := nod.Begin(" running %s with WINE, please wait...", cmdArg)
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

	absProtonPath, err := data.GeProtonCustomLatestReleasePath()
	if err != nil {
		return err
	}

	umuEnv := []string{
		"WINEPREFIX=" + absPrefixDir,
		"STORE=gog",
		"GAMEID=" + id,
		"PROTONPATH=" + absProtonPath,
	}

	cmd := exec.Command(absUmuRunPath, arg...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	cmd.Env = mergeEnv(umuEnv, env)

	return cmd.Run()
}

func linuxStartGogGamesLnk(id, langCode string, env []string, verbose bool, arg ...string) error {
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

		arg = append([]string{matches[0]}, arg...)

		relMatch, err := filepath.Rel(absPrefixDriveCDir, matches[0])
		if err != nil {
			return lsggla.EndWithError(err)
		}

		lsggla.EndWithResult("found %s", filepath.Join("C:", relMatch))

		return macOsWineRun(id, langCode, env, verbose, arg...)
	} else {
		return lsggla.EndWithError(errors.New("cannot locate suitable .lnk in the GOG Games folder"))
	}
}
