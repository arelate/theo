package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func DeletePrefixEnvHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	force := q.Has("force")

	return DeletePrefixEnv(ids, langCode, force)
}

func DeletePrefixEnv(ids []string, langCode string, force bool) error {

	dpea := nod.Begin("deleting prefix environment variables...")
	defer dpea.EndWithResult("done")

	if !force {
		dpea.EndWithResult("this operation requires force flag")
		return nil
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return dpea.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir, data.PrefixEnvProperty)
	if err != nil {
		return dpea.EndWithError(err)
	}

	prefixes := make([]string, 0, len(ids))
	for _, id := range ids {
		prefixes = append(prefixes, data.GetPrefixName(id, langCode))
	}

	if err := rdx.CutKeys(data.PrefixEnvProperty, prefixes...); err != nil {
		return dpea.EndWithError(err)
	}

	return nil
}
