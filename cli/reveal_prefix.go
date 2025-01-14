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
)

func RevealPrefixHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	return RevealPrefix(id, langCode)
}

func RevealPrefix(id, langCode string) error {

	rpa := nod.Begin("revealing prefix for %s...", id)
	defer rpa.EndWithResult("done")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return rpa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SlugProperty)
	if err != nil {
		return rpa.EndWithError(err)
	}

	prefixName, err := data.GetPrefixName(id, langCode, rdx)
	if err != nil {
		return rpa.EndWithError(err)
	}

	if prefixName == "" {
		rpa.EndWithResult("prefix for %s was not created", id)
		return nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(prefixName)
	if err != nil {
		return rpa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		rpa.EndWithResult("not found")
		return nil
	}

	absPrefixDriveCPath := filepath.Join(absPrefixDir, data.RelPfxDriveCDir)

	if err := currentOsReveal(absPrefixDriveCPath); err != nil {
		return rpa.EndWithError(err)
	}

	return nil

}
