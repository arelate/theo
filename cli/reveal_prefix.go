package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const relPrefixDriveCDir = "drive_c"

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

	rpa := nod.NewProgress("revealing prefix for %s...", strings.Join(ids, ","))
	defer rpa.EndWithResult("done")

	rpa.TotalInt(len(ids))

	for _, id := range ids {
		if err := revealProductPrefix(id, langCode); err != nil {
			return rpa.EndWithError(err)
		}

		rpa.Increment()
	}

	return nil
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

	absPrefixDriveCPath := filepath.Join(absPrefixDir, relPrefixDriveCDir)

	return currentOsReveal(absPrefixDriveCPath)
}
