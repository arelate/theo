package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	relCxAppDir             = "CrossOver.app"
	relCxBinDir             = "Contents/SharedSupport/CrossOver/bin"
	relCxBottleFilename     = "cxbottle"
	relCxBottleConfFilename = "cxbottle.conf"
	relWineFilename         = "wine"
)

const defaultCxBottleTemplate = "win10_64" // CrossOver.app/Contents/SharedSupport/CrossOver/share/crossover/bottle_templates

func macOsInitPrefix(absPrefixDir string, verbose bool) error {
	mipa := nod.Begin(" initializing %s prefix...", vangogh_integration.MacOS)
	defer mipa.Done()

	return macOsCreateCxBottle(absPrefixDir, defaultCxBottleTemplate, verbose)
}

func macOsWineExecTask(id string, et *execTask) error {

	mwra := nod.Begin(" running %s with WINE, please wait...", et.title)
	defer mwra.Done()

	if et.verbose && et.wrapper != "" {
		pwa := nod.Begin(" wrapper:")
		pwa.EndWithResult(et.wrapper)
	}

	if et.verbose {
		pea := nod.Begin(" exe:")
		pea.EndWithResult(et.exe)
	}

	if et.verbose && len(et.args) > 0 {
		paa := nod.Begin(" arg:")
		paa.EndWithResult(strings.Join(et.args, " "))
	}

	if et.verbose && len(et.env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(et.env, " "))
	}

	absCxBinDir, err := macOsGetAbsCxBinDir(nil)
	if err != nil {
		return err
	}

	absWineBinPath := filepath.Join(absCxBinDir, relWineFilename)

	if strings.HasSuffix(et.exe, ".lnk") {
		et.args = append([]string{"--start", et.exe}, et.args...)
	} else {

		switch et.wrapper {
		case "":
			et.args = append([]string{et.exe}, et.args...)
		default:
			et.args = append([]string{et.wrapper, et.exe}, et.args...)
		}
	}

	et.args = append([]string{"--bottle", et.prefix}, et.args...)

	if et.workDir != "" {
		et.args = append([]string{"--workdir", et.workDir}, et.args...)
	}

	cmd := exec.Command(absWineBinPath, et.args...)

	if et.workDir != "" {
		cmd.Dir = et.workDir
	}

	cmd.Env = et.env

	if et.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

func macOsGetAbsCxBinDir(rdx redux.Readable) (string, error) {

	if rdx == nil {
		var err error
		rdx, err = redux.NewReader(vangogh_integration.AbsReduxDir(), data.WineBinariesVersionsProperty)
		if err != nil {
			return "", err
		}
	} else {
		if err := rdx.MustHave(data.WineBinariesVersionsProperty); err != nil {
			return "", err
		}
	}

	var latestCxVersion string
	if lcxv, ok := rdx.GetLastVal(data.WineBinariesVersionsProperty, wine_integration.CrossOver); ok {
		latestCxVersion = lcxv
	}

	if latestCxVersion == "" {
		return "", errors.New("CrossOver version not found, please run setup-wine")
	}

	binariesRuntimesDir := camino.GetRel(vangogh_integration.Runtimes, vangogh_integration.Binaries)

	absCrossOverBinDir := filepath.Join(binariesRuntimesDir, camino.Sanitize(wine_integration.CrossOver), latestCxVersion, relCxAppDir, relCxBinDir)
	if _, err := os.Stat(absCrossOverBinDir); err == nil {
		return absCrossOverBinDir, nil
	}

	return "", os.ErrNotExist
}

func macOsCreateCxBottle(absPrefixDir string, template string, verbose bool) error {

	if template == "" {
		template = defaultCxBottleTemplate
	}

	absCxBinDir, err := macOsGetAbsCxBinDir(nil)
	if err != nil {
		return err
	}

	absCxBottlePath := filepath.Join(absCxBinDir, relCxBottleFilename)

	// cxbottle --create returns error when bottle already exists
	if _, err = os.Stat(absPrefixDir); err == nil {

		// if a prefix exists, but is missing cxbottle.conf - there will be an error
		absCxBottleConfPath := filepath.Join(absPrefixDir, relCxBottleConfFilename)
		if _, err = os.Stat(absCxBottleConfPath); os.IsNotExist(err) {
			if _, err = os.Create(absCxBottleConfPath); err != nil {
				return err
			}
		}

		return nil
	}

	cmd := exec.Command(absCxBottlePath, "--bottle", absPrefixDir, "--create", "--template", template)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}
