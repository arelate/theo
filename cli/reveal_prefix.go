package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
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

	if len(ids) == 1 {
		return revealProductPrefix(ids[0], langCode)
	} else {
		if absPrefixesDir, err := pathways.GetAbsDir(data.Prefixes); err != nil {
			return rpa.EndWithError(err)
		} else {
			return currentOsReveal(absPrefixesDir)
		}
	}
}

func revealProductPrefix(id, langCode string) error {

	rppa := nod.Begin(" revealing prefix for %s...", id)
	defer rppa.EndWithResult("done")

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
	if err != nil {
		return rppa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		rppa.EndWithResult("not found")
		return nil
	}

	absPrefixDriveCPath := filepath.Join(absPrefixDir, relPrefixDriveCDir, gogGamesDir)

	return currentOsReveal(absPrefixDriveCPath)
}
