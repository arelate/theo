package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func DeletePrefixExePathHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	force := q.Has("force")

	return DeletePrefixExePath(ids, langCode, force)
}

func DeletePrefixExePath(ids []string, langCode string, force bool) error {

	dpepa := nod.Begin("deleting prefix exe paths...")
	defer dpepa.EndWithResult("done")

	if !force {
		dpepa.EndWithResult("this operation requires force flag")
		return nil
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return dpepa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir, data.PrefixExePathProperty)
	if err != nil {
		return dpepa.EndWithError(err)
	}

	prefixes := make([]string, 0, len(ids))
	for _, id := range ids {
		prefixes = append(prefixes, data.GetPrefixName(id, langCode))
	}

	if err := rdx.CutKeys(data.PrefixExePathProperty, prefixes...); err != nil {
		return dpepa.EndWithError(err)
	}

	return nil
}
