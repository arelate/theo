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
	"path/filepath"
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
