package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
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

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty)
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

func removeProductPrefix(id, langCode string, rdx redux.Readable, archive, force bool) error {
	rppa := nod.Begin(" removing prefix for %s...", id)
	defer rppa.EndWithResult("done")

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return rppa.EndWithError(err)
	}

	if _, err = os.Stat(absPrefixDir); os.IsNotExist(err) {
		rppa.EndWithResult("not present")
		return nil
	}

	if archive {
		if err = ArchivePrefix(langCode, id); err != nil {
			return rppa.EndWithError(err)
		}
	}

	if !force {
		rppa.EndWithResult("found prefix, use -force to remove")
		return nil
	}

	// TODO: Currently, this removes all files in the prefix, including configuration and save games,
	// that would be typically stored in some app data directory outside of the main game installation.
	// Sooner rather than later this should be replaced by something more nuanced that preserves those files
	return os.RemoveAll(absPrefixDir)
}
