package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"strings"
)

func WineRunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	exePath := q.Get("exe-path")
	verbose := q.Has("verbose")
	env := make([]string, 0)
	if q.Has("env") {
		env = strings.Split(q.Get("env"), ",")
	}

	return WineRun(id, langCode, exePath, env, verbose)
}

func WineRun(id string, langCode string, exePath string, env []string, verbose bool) error {

	wra := nod.Begin("running %s version with WINE...", vangogh_integration.Windows)
	defer wra.EndWithResult("done")

	vangogh_integration.PrintParams([]string{id},
		[]vangogh_integration.OperatingSystem{vangogh_integration.Windows},
		[]string{langCode},
		nil,
		false)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return wra.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir, data.PrefixEnvProperty)
	if err != nil {
		return wra.EndWithError(err)
	}

	prefixName := data.GetPrefixName(id, langCode)

	prefixEnv, _ := rdx.GetAllValues(data.PrefixEnvProperty, prefixName)
	prefixEnv = mergeEnv(prefixEnv, env)

	switch data.CurrentOS() {
	case vangogh_integration.MacOS:
		if exePath != "" {
			if err := macOsWineRun(id, langCode, prefixEnv, verbose, exePath); err != nil {
				return err
			}
		} else if err := macOsStartGogGamesLnk(id, langCode, prefixEnv, verbose); err != nil {
			return err
		}
	default:
		panic("not implemented")
	}

	return nil
}
