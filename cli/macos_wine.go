package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	absDefaultApplicationsDir = "/Applications"
	relCxAppDir               = "CrossOver.app"
	relCxBinDir               = "Contents/SharedSupport/CrossOver/bin"
	relCxBottleFilename       = "cxbottle"
	relWineFilename           = "wine"
)

const defaultCxBottleTemplate = "win10_64" // CrossOver.app/Contents/SharedSupport/CrossOver/share/crossover/bottle_templates

const gogInstallationLnkGlob = "GOG Games/*/*.lnk"

type (
	wineRunFunc        func(id, langCode string, env []string, verbose, force bool, exePath string, arg ...string) error
	wineInitPrefixFunc func(id, langCode string, verbose bool) error
)

func macOsInitPrefix(id, langCode string, verbose bool) error {
	mipa := nod.Begin(" initializing %s prefix...", vangogh_integration.MacOS)
	defer mipa.EndWithResult("done")

	return macOsCreateCxBottle(id, langCode, defaultCxBottleTemplate, verbose)
}

func macOsWineRun(id, langCode string, env []string, verbose, force bool, exePath string, arg ...string) error {

	_, exeFilename := filepath.Split(exePath)

	mwra := nod.Begin(" running %s with WINE, please wait...", exeFilename)
	defer mwra.EndWithResult("done")

	if verbose && len(env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(env, " "))
	}

	absCxBinDir, err := macOsGetAbsCxBinDir()
	if err != nil {
		return err
	}

	absWineBinPath := filepath.Join(absCxBinDir, relWineFilename)

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
	if err != nil {
		return err
	}

	if strings.HasSuffix(exePath, ".lnk") {
		arg = append([]string{"--start", exePath}, arg...)
	} else {
		arg = append([]string{exePath}, arg...)
	}

	arg = append([]string{"--bottle", absPrefixDir}, arg...)

	cmd := exec.Command(absWineBinPath, arg...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	for _, e := range env {
		cmd.Env = append(cmd.Env, e)
	}

	return cmd.Run()
}

func getPrefixGogGamesLnk(id, langCode string) (string, error) {

	msggla := nod.Begin(" locating default .lnk in the install folder for %s...", id)
	defer msggla.EndWithResult("done")

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
	if err != nil {
		return "", msggla.EndWithError(err)
	}

	absPrefixDriveCDir := filepath.Join(absPrefixDir, relPrefixDriveCDir)

	matches, err := filepath.Glob(filepath.Join(absPrefixDriveCDir, gogInstallationLnkGlob))
	if err != nil {
		return "", msggla.EndWithError(err)
	}

	if len(matches) == 1 {

		relMatch, err := filepath.Rel(absPrefixDriveCDir, matches[0])
		if err != nil {
			return "", msggla.EndWithError(err)
		}
		msggla.EndWithResult("found %s", filepath.Join("C:", relMatch))

		return matches[0], nil
	} else {
		return "", msggla.EndWithError(errors.New("cannot locate suitable .lnk in the GOG Games folder"))
	}
}

func macOsGetAbsCxBinDir(appDirs ...string) (string, error) {
	if len(appDirs) == 0 {
		appDirs = append(appDirs, absDefaultApplicationsDir)
	}

	for _, appDir := range appDirs {
		absCrossOverBinDir := filepath.Join(appDir, relCxAppDir, relCxBinDir)
		if _, err := os.Stat(absCrossOverBinDir); err == nil {
			return absCrossOverBinDir, nil
		}
	}

	return "", os.ErrNotExist
}

func macOsCreateCxBottle(id, langCode string, template string, verbose bool) error {

	if template == "" {
		template = defaultCxBottleTemplate
	}

	absCxBinDir, err := macOsGetAbsCxBinDir()
	if err != nil {
		return err
	}

	absCxBottlePath := filepath.Join(absCxBinDir, relCxBottleFilename)

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
	if err != nil {
		return err
	}

	cmd := exec.Command(absCxBottlePath, "--bottle", absPrefixDir, "--create", "--template", template)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}
