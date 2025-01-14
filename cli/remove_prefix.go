package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"strings"
)

func RemovePrefixHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	archive := !q.Has("no-archive")
	force := q.Has("force")

	return RemovePrefix(langCode, archive, force, ids...)
}

func RemovePrefix(langCode string, archive, force bool, ids ...string) error {

	rpa := nod.NewProgress("removing prefixes for %s...", strings.Join(ids, ","))
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
		if err := removeProductPrefix(id, langCode, rdx, archive, force); err != nil {
			return rpa.EndWithError(err)
		}

		rpa.Increment()
	}

	return nil
}

func removeProductPrefix(id, langCode string, rdx kevlar.ReadableRedux, archive, force bool) error {
	rppa := nod.Begin(" removing prefix for %s...", id)
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
		rppa.EndWithResult("not present")
		return nil
	}

	if archive {
		if err := ArchivePrefix(langCode, id); err != nil {
			return rppa.EndWithError(err)
		}
	} else {
		// do nothing
	}

	if !force {
		rppa.EndWithResult("found prefix, use -force to remove")
		return nil
	}

	return os.RemoveAll(absPrefixDir)
}
