package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

func RunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_local_data.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_local_data.LanguageCodeProperty) {
		langCode = q.Get(vangogh_local_data.LanguageCodeProperty)
	}

	return Run(id, langCode)
}

func Run(id string, langCode string) error {

	ra := nod.NewProgress("running product %s...", id)
	defer ra.EndWithResult("done")

	currentOs := []vangogh_local_data.OperatingSystem{data.CurrentOS()}
	langCodes := []string{langCode}

	vangogh_local_data.PrintParams([]string{id}, currentOs, langCodes, nil, true)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return ra.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties, data.BundleNameProperty)
	if err != nil {
		return ra.EndWithError(err)
	}

	return currentOsRunApp(id, langCode, rdx)
}

func currentOsRunApp(id, langCode string, rdx kevlar.ReadableRedux) error {

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	bundleName, _ := rdx.GetLastVal(data.BundleNameProperty, id)
	absBundlePath := filepath.Join(installedAppsDir, data.OsLangCodeDir(data.CurrentOS(), langCode), bundleName)

	if _, err := os.Stat(absBundlePath); err != nil {
		return err
	}

	if err := currentOsExecute(absBundlePath); err != nil {
		return err
	}

	return nil
}

func currentOsExecute(path string) error {
	switch data.CurrentOS() {
	case vangogh_local_data.MacOS:
		return macOsExecute(path)
	case vangogh_local_data.Windows:
		return windowsExecute(path)
	case vangogh_local_data.Linux:
		return linuxExecute(path)
	default:
		return errors.New("cannot reveal on unknown operating system")
	}
}

func macOsExecute(path string) error {

	path = macOsLocateAppBundle(path)

	cmd := exec.Command("open", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func windowsExecute(path string) error {
	return errors.New("support for running executables on Windows is not implemented")
}

func linuxExecute(path string) error {
	// TODO: investigate what we need to specify on Linux to be able to run apps
	return errors.New("support for running executables on Linux is not implemented")
}
