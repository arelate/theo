package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func UmuRunHandler(u *url.URL) error {

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
	force := q.Has("force")

	return UmuRun(id, langCode, env, verbose, force)
}

func UmuRun(id string, langCode string, env []string, verbose, force bool) error {

	ura := nod.NewProgress("running product %s with umu-launcher...", id)
	defer ura.EndWithResult("done")

	if data.CurrentOs() != vangogh_integration.Linux {
		ura.EndWithResult("umu-launcher is only supported on %s", vangogh_integration.Linux)
		return nil
	}

	currentOs := []vangogh_integration.OperatingSystem{data.CurrentOs()}
	langCodes := []string{langCode}

	vangogh_integration.PrintParams([]string{id}, currentOs, langCodes, nil, true)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return ura.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties, data.BundleNameProperty)
	if err != nil {
		return ura.EndWithError(err)
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	bundleName, _ := rdx.GetLastVal(data.BundleNameProperty, id)
	absBundlePath := filepath.Join(installedAppsDir, data.OsLangCodeDir(data.CurrentOs(), langCode), bundleName)

	if _, err := os.Stat(absBundlePath); err != nil {
		return err
	}

	startSh := linuxLocateStartSh(absBundlePath)

	return linuxUmuRun(id, startSh, env, verbose, force)
}
