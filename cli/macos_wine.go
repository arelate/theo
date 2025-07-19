package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	relCxAppDir             = "CrossOver.app"
	relCxBinDir             = "Contents/SharedSupport/CrossOver/bin"
	relCxBottleFilename     = "cxbottle"
	relCxBottleConfFilename = "cxbottle.conf"
	relWineFilename         = "wine"
)

const defaultCxBottleTemplate = "win10_64" // CrossOver.app/Contents/SharedSupport/CrossOver/share/crossover/bottle_templates

type (
	wineRunFunc func(id, langCode string, rdx redux.Readable, et *execTask, force bool) error
)

func macOsInitPrefix(id, langCode string, rdx redux.Readable, verbose bool) error {
	mipa := nod.Begin(" initializing %s prefix...", vangogh_integration.MacOS)
	defer mipa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	return macOsCreateCxBottle(id, langCode, rdx, defaultCxBottleTemplate, verbose)
}

func macOsWineRun(id, langCode string, rdx redux.Readable, et *execTask, force bool) error {

	_, exeFilename := filepath.Split(et.exe)

	mwra := nod.Begin(" running %s with WINE, please wait...", exeFilename)
	defer mwra.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	if et.verbose && len(et.env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(et.env, " "))
	}

	absCxBinDir, err := macOsGetAbsCxBinDir(rdx)
	if err != nil {
		return err
	}

	absWineBinPath := filepath.Join(absCxBinDir, relWineFilename)

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	if strings.HasSuffix(et.exe, ".lnk") {
		et.args = append([]string{"--start", et.exe}, et.args...)
	} else {
		et.args = append([]string{et.exe}, et.args...)
	}

	et.args = append([]string{"--bottle", absPrefixDir}, et.args...)

	cmd := exec.Command(absWineBinPath, et.args...)

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

func macOsWineRunExecTask(et *execTask, rdx redux.Readable) error {

	mwra := nod.Begin(" running %s with WINE, please wait...", et.name)
	defer mwra.Done()

	if et.verbose && len(et.env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(et.env, " "))
	}

	absCxBinDir, err := macOsGetAbsCxBinDir(rdx)
	if err != nil {
		return err
	}

	absWineBinPath := filepath.Join(absCxBinDir, relWineFilename)

	if strings.HasSuffix(et.exe, ".lnk") {
		et.args = append([]string{"--start", et.exe}, et.args...)
	} else {
		et.args = append([]string{et.exe}, et.args...)
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

	if err := rdx.MustHave(data.WineBinariesVersionsProperty); err != nil {
		return "", err
	}

	var latestCxVersion string
	if lcxv, ok := rdx.GetLastVal(data.WineBinariesVersionsProperty, vangogh_integration.CrossOver); ok {
		latestCxVersion = lcxv
	}

	if latestCxVersion == "" {
		return "", errors.New("CrossOver version not found, please run setup-wine")
	}

	wineBinaries, err := pathways.GetAbsRelDir(data.WineBinaries)
	if err != nil {
		return "", err
	}

	absCrossOverBinDir := filepath.Join(wineBinaries, busan.Sanitize(vangogh_integration.CrossOver), latestCxVersion, relCxAppDir, relCxBinDir)
	if _, err = os.Stat(absCrossOverBinDir); err == nil {
		return absCrossOverBinDir, nil
	}

	return "", os.ErrNotExist
}

func macOsCreateCxBottle(id, langCode string, rdx redux.Readable, template string, verbose bool) error {

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	if template == "" {
		template = defaultCxBottleTemplate
	}

	absCxBinDir, err := macOsGetAbsCxBinDir(rdx)
	if err != nil {
		return err
	}

	absCxBottlePath := filepath.Join(absCxBinDir, relCxBottleFilename)

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

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
