package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	linuxStartShFilename = "start.sh"
)

func RunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	return Run(id, langCode)
}

func Run(id string, langCode string) error {

	ra := nod.NewProgress("running product %s...", id)
	defer ra.EndWithResult("done")

	currentOs := []vangogh_integration.OperatingSystem{data.CurrentOS()}
	langCodes := []string{langCode}

	vangogh_integration.PrintParams([]string{id}, currentOs, langCodes, nil, true)

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
	case vangogh_integration.MacOS:
		return macOsExecute(path)
	case vangogh_integration.Windows:
		return windowsExecute(path)
	case vangogh_integration.Linux:
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

	startShPath := linuxLocateStartSh(path)

	cmd := exec.Command(startShPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func linuxLocateStartSh(path string) string {
	if strings.HasSuffix(path, linuxStartShFilename) {
		return path
	}

	absStartShPath := filepath.Join(path, linuxStartShFilename)
	if _, err := os.Stat(absStartShPath); err == nil {
		return absStartShPath
	} else if os.IsNotExist(err) {
		if matches, err := filepath.Glob(filepath.Join(path, "*", linuxStartShFilename)); err == nil && len(matches) > 0 {
			return matches[0]
		}
	}

	return path
}
