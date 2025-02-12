package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"os"
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
	var env []string
	if q.Has("env") {
		env = strings.Split(q.Get("env"), ",")
	}
	verbose := q.Has("verbose")

	return Run(id, langCode, env, verbose)
}

func Run(id string, langCode string, env []string, verbose bool) error {

	ra := nod.NewProgress("running product %s...", id)
	defer ra.EndWithResult("done")

	currentOs := []vangogh_integration.OperatingSystem{data.CurrentOs()}
	langCodes := []string{langCode}

	vangogh_integration.PrintParams([]string{id}, currentOs, langCodes, nil, true)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir, data.ServerConnectionProperties, data.BundleNameProperty)
	if err != nil {
		return err
	}

	return currentOsRunApp(id, langCode, rdx, env, verbose)
}

func currentOsRunApp(id, langCode string, rdx redux.Readable, env []string, verbose bool) error {

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	bundleName, _ := rdx.GetLastVal(data.BundleNameProperty, id)
	absBundlePath := filepath.Join(installedAppsDir, data.OsLangCode(data.CurrentOs(), langCode), bundleName)

	if _, err := os.Stat(absBundlePath); err != nil {
		return err
	}

	if err := currentOsExecute(absBundlePath, env, verbose); err != nil {
		return err
	}

	return nil
}

func currentOsExecute(path string, env []string, verbose bool) error {
	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		return macOsExecute(path, env, verbose)
	case vangogh_integration.Windows:
		return windowsExecute(path, env, verbose)
	case vangogh_integration.Linux:
		return linuxExecute(path, env, verbose)
	default:
		return errors.New("cannot reveal on unknown operating system")
	}
}
