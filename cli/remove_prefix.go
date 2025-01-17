package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
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

	rpa.TotalInt(len(ids))

	for _, id := range ids {
		if err := removeProductPrefix(id, langCode, archive, force); err != nil {
			return rpa.EndWithError(err)
		}

		rpa.Increment()
	}

	return nil
}

func removeProductPrefix(id, langCode string, archive, force bool) error {
	rppa := nod.Begin(" removing prefix for %s...", id)
	defer rppa.EndWithResult("done")

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
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
	}

	if !force {
		rppa.EndWithResult("found prefix, use -force to remove")
		return nil
	}

	return os.RemoveAll(absPrefixDir)
}
