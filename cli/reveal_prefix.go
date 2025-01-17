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

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return rpa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SlugProperty)
	if err != nil {
		return rpa.EndWithError(err)
	}

	rpa.TotalInt(len(ids))

	for _, id := range ids {
		if err := revealProductPrefix(id, langCode, rdx); err != nil {
			return rpa.EndWithError(err)
		}

		rpa.Increment()
	}

	return nil
}

func revealProductPrefix(id, langCode string, rdx kevlar.ReadableRedux) error {

	rppa := nod.Begin(" revealing prefix for %s...", id)
	defer rppa.EndWithResult("done")

	prefixName, err := data.GetPrefixName(id, langCode, rdx)
	if err != nil {
		return rppa.EndWithError(err)
	}

	if prefixName == "" {
		rppa.EndWithResult("prefix for %s was not created", id)
		return nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(prefixName)
	if err != nil {
		return rppa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		rppa.EndWithResult("not found")
		return nil
	}

	absPrefixDriveCPath := filepath.Join(absPrefixDir, data.RelPrefixDriveCDir)

	return currentOsReveal(absPrefixDriveCPath)
}
