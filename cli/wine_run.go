package cli

import (
	"errors"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

const (
	gogGameInstallDir      = gogGamesDir + "/*"
	gogInstallationLnkGlob = gogGamesDir + "/*/*.lnk"
	gogGameInfoGlob        = gogGamesDir + "/*/goggame-{id}.info"
)

func WineRunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	et := &execTask{
		exe:     q.Get("exe-path"),
		workDir: q.Get("working-dir"),
		verbose: q.Has("verbose"),
	}
	if q.Has("env") {
		et.env = strings.Split(q.Get("env"), ",")
	}

	force := q.Has("force")

	return WineRun(id, langCode, et, force)
}

func WineRun(id string, langCode string, et *execTask, force bool) error {

	wra := nod.Begin("running %s version with WINE...", vangogh_integration.Windows)
	defer wra.Done()

	vangogh_integration.PrintParams([]string{id},
		[]vangogh_integration.OperatingSystem{vangogh_integration.Windows},
		[]string{langCode},
		nil,
		false)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	if err = checkProductType(id, rdx, force); err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	prefixEnv, _ := rdx.GetAllValues(data.PrefixEnvProperty, path.Join(prefixName, langCode))
	et.env = mergeEnv(prefixEnv, et.env)

	if et.exe == "" {
		et.exe, err = getExePath(id, langCode, rdx)
		if err != nil {
			return err
		}
	}

	if et.workDir == "" {
		et.workDir, err = findGogGameInstallPath(id, langCode, rdx)
		if err != nil {
			return err
		}
	}

	if _, err = os.Stat(et.exe); err != nil {
		return err
	}

	if err = setLastRunDate(rdx, id); err != nil {
		return err
	}

	var currentOsWineRun wineRunFunc

	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		currentOsWineRun = macOsWineRun
	case vangogh_integration.Linux:
		currentOsWineRun = linuxProtonRun
	default:
		return errors.New("wine-run: unsupported operating system")
	}

	return currentOsWineRun(id, langCode, rdx, et, force)
}
