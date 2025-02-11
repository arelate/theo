package cli

import (
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
	relPrefixDriveCDir = "drive_c"
	gogGamesDir        = "GOG Games"
)

func RevealPrefixHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	return RevealPrefix(langCode, ids...)
}

func RevealPrefix(langCode string, ids ...string) error {

	rpa := nod.Begin("revealing prefix for %s...", strings.Join(ids, ","))
	defer rpa.EndWithResult("done")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return rpa.EndWithError(err)
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty)
	if err != nil {
		return rpa.EndWithError(err)
	}

	if len(ids) == 1 {
		return revealProductPrefix(ids[0], langCode, rdx)
	} else {

		if installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps); err != nil {
			return err
		} else {
			osLang := strings.Join([]string{vangogh_integration.Windows.String(), langCode}, "-")
			return currentOsReveal(filepath.Join(installedAppsDir, osLang))
		}
	}
}

func revealProductPrefix(id, langCode string, rdx redux.Readable) error {

	rppa := nod.Begin(" revealing prefix for %s...", id)
	defer rppa.EndWithResult("done")

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return rppa.EndWithError(err)
	}

	if _, err = os.Stat(absPrefixDir); os.IsNotExist(err) {
		rppa.EndWithResult("not found")
		return nil
	}

	absPrefixDriveCPath := filepath.Join(absPrefixDir, relPrefixDriveCDir, gogGamesDir)

	return currentOsReveal(absPrefixDriveCPath)
}
