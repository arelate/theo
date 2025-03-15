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
	"path"
	"path/filepath"
	"strings"
)

func WineRunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	env := make([]string, 0)
	if q.Has("env") {
		env = strings.Split(q.Get("env"), ",")
	}
	exePath := q.Get("exe-path")
	verbose := q.Has("verbose")
	force := q.Has("force")

	return WineRun(id, langCode, exePath, env, verbose, force)
}

func WineRun(id string, langCode string, exePath string, env []string, verbose, force bool) error {

	wra := nod.Begin("running %s version with WINE...", vangogh_integration.Windows)
	defer wra.Done()

	vangogh_integration.PrintParams([]string{id},
		[]vangogh_integration.OperatingSystem{vangogh_integration.Windows},
		[]string{langCode},
		nil,
		false)
	if err := resolveProductTitles(id); err != nil {
		return err
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.PrefixEnvProperty, data.PrefixExePathProperty)
	if err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	prefixEnv, _ := rdx.GetAllValues(data.PrefixEnvProperty, path.Join(prefixName, langCode))
	prefixEnv = mergeEnv(prefixEnv, env)

	if ep, ok := rdx.GetLastVal(data.PrefixExePathProperty, prefixName); ok && ep != "" {
		absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
		if err != nil {
			return err
		}

		exePath = filepath.Join(absPrefixDir, relPrefixDriveCDir, ep)
	}

	if exePath == "" {
		exePath, err = getPrefixGogGamesLnk(id, langCode, rdx)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(exePath); err != nil {
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

	return currentOsWineRun(id, langCode, rdx, prefixEnv, verbose, force, exePath)
}
