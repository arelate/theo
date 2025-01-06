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

	var id string
	if ids := Ids(u); len(ids) >= 1 {
		id = ids[0]
	}

	langCode := defaultLangCode
	if u.Query().Has(vangogh_local_data.LanguageCodeProperty) {
		langCode = u.Query().Get(vangogh_local_data.LanguageCodeProperty)
	}

	return Run(id, langCode)
}

func Run(id string, langCode string) error {

	ra := nod.NewProgress("running product %s...", id)
	defer ra.EndWithResult("done")

	currentOs := []vangogh_local_data.OperatingSystem{CurrentOS()}
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
	productInstalledAppDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(CurrentOS(), langCode), bundleName)

	if err := currentOsExecute(productInstalledAppDir); err != nil {
		return err
	}

	return nil
}

func currentOsExecute(path string) error {
	switch CurrentOS() {
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
	cmd := exec.Command("open", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func windowsExecute(path string) error {
	return errors.New("support for running executables on Windows is not implemented")
}

func linuxExecute(path string) error {
	cmd := exec.Command("xdg-open", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
