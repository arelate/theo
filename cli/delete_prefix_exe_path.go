package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path"
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
	defer dpepa.Done()

	if err := resolveProductTitles(ids...); err != nil {
		return err
	}

	if !force {
		dpepa.EndWithResult("this operation requires force flag")
		return nil
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.PrefixExePathProperty)
	if err != nil {
		return err
	}

	prefixes := make([]string, 0, len(ids))
	for _, id := range ids {
		prefixName, err := data.GetPrefixName(id, rdx)
		if err != nil {
			return err
		}

		prefixes = append(prefixes, path.Join(prefixName, langCode))
	}

	if err = rdx.CutKeys(data.PrefixExePathProperty, prefixes...); err != nil {
		return err
	}

	return nil
}
