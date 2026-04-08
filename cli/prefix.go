package cli

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func PrefixHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	var langCode string
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	ii := &InstallInfo{
		OperatingSystem: vangogh_integration.Windows,
		LangCode:        langCode,
		force:           q.Has("force"),
	}

	et := new(execTask{
		exe:     q.Get("exe"),
		verbose: q.Has("verbose"),
	})

	if q.Has("env") {
		et.env = strings.Split(q.Get("env"), ",")
	}

	if q.Has("arg") {
		et.args = strings.Split(q.Get("arg"), ",")
	}

	mod := q.Get("mod")
	program := q.Get("program")
	wineBinary := q.Get("install-wine-binary")

	return Prefix(id, ii,
		mod, program, wineBinary,
		et)
}

func Prefix(id string,
	request *InstallInfo,
	mod, program, wineBinary string,
	et *execTask) error {

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	ii, err := matchInstalledInfo(id, request, rdx)
	if err != nil {
		return err
	}

	absPrefixDir, err := data.AbsPrefixDir(id, ii.Origin, rdx)
	if err != nil {
		return err
	}

	et.prefix = absPrefixDir

	if et.exe != "" {
		et.title = filepath.Base(et.exe)
		return osExec(id, vangogh_integration.Windows, et)
	}

	if mod != "" {

		switch mod {
		case prefixModEnableRetina:
			if err = prefixModRetina(id, ii.Origin, false, rdx, et.verbose, ii.force); err != nil {
				return err
			}
		case prefixModDisableRetina:
			if err = prefixModRetina(id, ii.Origin, true, rdx, et.verbose, ii.force); err != nil {
				return err
			}
		}

	}

	if program != "" {

		if !slices.Contains(wine_integration.WinePrograms(), program) {
			return errors.New("unknown prefix WINE program " + program)
		}

		et.title = program
		et.exe = program

		if err = osExec(id, vangogh_integration.Windows, et); err != nil {
			return err
		}

	}

	if wineBinary != "" {
		if err = prefixInstallBinary(id, wineBinary, absPrefixDir, et); err != nil {
			return err
		}
	}

	return nil
}

func prefixInstallBinary(id string, wineBinary string, absPrefixDir string, et *execTask) error {

	if !slices.Contains(wine_integration.WineBinariesCodes(), wineBinary) {
		return errors.New("unknown WINE binary " + wineBinary)
	}

	var requestedWineBinary *wine_integration.Binary
	for _, binary := range wine_integration.OsWineBinaries {
		if binary.OS == vangogh_integration.Windows && binary.Code == wineBinary {
			requestedWineBinary = &binary
		}
	}

	if requestedWineBinary == nil {
		return errors.New("no match for WINE binary code " + wineBinary)
	}

	// This would only support direct download sources.
	// Currently all coded WINE binaries are direct download sources, so this if fine for now.
	wbFilename := path.Base(requestedWineBinary.DownloadUrl)

	var wineDownloadsDir string
	wineDownloadsDir = data.Pwd.AbsRelDirPath(data.BinDownloads, data.Wine)

	et.title = requestedWineBinary.String()
	et.exe = filepath.Join(wineDownloadsDir, wbFilename)

	if args, ok := wine_integration.WineBinariesCodesArgs[wineBinary]; ok {
		et.args = args
	}

	if _, err := os.Stat(et.exe); os.IsNotExist(err) {
		return errors.New("matched WINE binary not found, use setup-wine to download")
	}

	switch wineBinary {
	case wine_integration.DxEndUserRuntimeCode:

		originalName := et.title

		dxSetupDir, _ := filepath.Split(wine_integration.DxSetupPath)
		absDxUnpackDir := filepath.Join(absPrefixDir, prefixRelDriveCDir, dxSetupDir)

		if _, err := os.Stat(absDxUnpackDir); err == nil {
			if err = os.RemoveAll(absDxUnpackDir); err != nil {
				return err
			}
		}

		et.title = "extract: " + originalName

		// extract to {prefix}/drive_c/DirectX
		if err := osExec(id, vangogh_integration.Windows, et); err != nil {
			return err
		}

		// Set execTask to run DXSetup.exe on func exit
		et.exe = filepath.Join(absPrefixDir, prefixRelDriveCDir, wine_integration.DxSetupPath)
		et.args = wine_integration.DxSetupArgs
		et.title = "install: " + originalName

	default:
		// do nothing
	}

	return osExec(id, vangogh_integration.Windows, et)
}

func prefixModRetina(id string, origin data.Origin, revert bool, rdx redux.Writeable, verbose, force bool) error {

	mpa := nod.Begin("modding retina in prefix for %s...", id)
	defer mpa.Done()

	if data.CurrentOs() != vangogh_integration.MacOS {
		mpa.EndWithResult("retina prefix mod is only applicable to %s", vangogh_integration.MacOS)
		return nil
	}

	if err := rdx.MustHave(vangogh_integration.TitleProperty); err != nil {
		return err
	}

	absPrefixDir, err := data.AbsPrefixDir(id, origin, rdx)
	if err != nil {
		return err
	}

	absDriveCroot := filepath.Join(absPrefixDir, prefixRelDriveCDir)

	regFilename := retinaOnFilename
	regContent := retinaOnReg
	if revert {
		regFilename = retinaOffFilename
		regContent = retinaOffReg
	}

	absRegPath := filepath.Join(absDriveCroot, regFilename)
	if _, err = os.Stat(absRegPath); os.IsNotExist(err) || (err == nil && force) {
		if err = createRegFile(absRegPath, regContent); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	et := &execTask{
		exe:     regeditBin,
		workDir: absDriveCroot,
		prefix:  absPrefixDir,
		args:    []string{absRegPath},
		verbose: verbose,
	}

	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		if err = macOsWineExecTask(id, et); err != nil {
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

	if _, err = io.Copy(regFile, bytes.NewReader(content)); err != nil {
		return err
	}

	return nil
}
