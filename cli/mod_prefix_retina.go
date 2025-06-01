package cli

import (
	"bytes"
	_ "embed"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"net/url"
	"os"
	"path/filepath"
)

const (
	retinaOnFilename  = "retina_on.reg"
	retinaOffFilename = "retina_off.reg"
)

const regeditBin = "regedit"

var (
	//go:embed "registry/retina_on.reg"
	retinaOnReg []byte
	//go:embed "registry/retina_off.reg"
	retinaOffReg []byte
)

func ModPrefixRetinaHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	revert := q.Has("revert")
	verbose := q.Has("verbose")
	force := q.Has("force")

	return ModPrefixRetina(id, langCode, revert, verbose, force)
}

func ModPrefixRetina(id, langCode string, revert, verbose, force bool) error {

	mpa := nod.Begin("modding retina in prefix for %s...", id)
	defer mpa.Done()

	if data.CurrentOs() != vangogh_integration.MacOS {
		mpa.EndWithResult("retina prefix mod is only applicable to %s", vangogh_integration.MacOS)
		return nil
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, vangogh_integration.SlugProperty, data.PrefixEnvProperty, data.PrefixExePathProperty)
	if err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	absDriveCroot := filepath.Join(absPrefixDir, relPrefixDriveCDir)

	regFilename := retinaOnFilename
	regContent := retinaOnReg
	if revert {
		regFilename = retinaOffFilename
		regContent = retinaOffReg
	}

	absRegPath := filepath.Join(absDriveCroot, regFilename)
	if _, err := os.Stat(absRegPath); os.IsNotExist(err) || (err == nil && force) {
		if err := createRegFile(absRegPath, regContent); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	et := &execTask{
		exe:     regeditBin,
		workDir: absDriveCroot,
		args:    []string{absRegPath},
		verbose: verbose,
	}

	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		if err := macOsWineRun(id, langCode, rdx, et, force); err != nil {
			return err
		}
	default:
		// do nothing
		return nil
	}
	return nil
}

func createRegFile(absPath string, content []byte) error {

	regFile, err := os.Create(absPath)
	if err != nil {
		return err
	}
	defer regFile.Close()

	if _, err := io.Copy(regFile, bytes.NewReader(content)); err != nil {
		return err
	}

	return nil
}
