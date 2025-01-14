package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/backups"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func ArchivePrefixHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	return ArchivePrefix(langCode, ids...)
}

func ArchivePrefix(langCode string, ids ...string) error {

	apa := nod.NewProgress("archiving prefixes for %s...", strings.Join(ids, ","))
	defer apa.EndWithResult("done")

	vangogh_integration.PrintParams(ids, nil, []string{langCode}, nil, true)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return apa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SlugProperty)
	if err != nil {
		return apa.EndWithError(err)
	}

	apa.TotalInt(len(ids))

	for _, id := range ids {

		if err := archiveProductPrefix(id, langCode, rdx); err != nil {
			return apa.EndWithError(err)
		}

		apa.Increment()
	}

	return nil

}

func archiveProductPrefix(id, langCode string, rdx kevlar.ReadableRedux) error {

	appa := nod.Begin(" archiving prefix for %s...", id)
	defer appa.EndWithResult("done")

	prefixArchiveDir, err := pathways.GetAbsRelDir(data.PrefixArchive)
	if err != nil {
		return appa.EndWithError(err)
	}

	prefixName, err := data.GetPrefixName(id, langCode, rdx)
	if err != nil {
		return appa.EndWithError(err)
	}

	if prefixName == "" {
		appa.EndWithResult("prefix for %s was not created", id)
		return nil
	}

	absPrefixNameArchiveDir := filepath.Join(prefixArchiveDir, prefixName)

	absPrefixDir, err := data.GetAbsPrefixDir(prefixName)
	if err != nil {
		return appa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixNameArchiveDir); err != nil {
		if err := os.MkdirAll(absPrefixNameArchiveDir, 0755); err != nil {
			return appa.EndWithError(err)
		}
	}

	if err := backups.Compress(absPrefixDir, absPrefixNameArchiveDir); err != nil {
		return appa.EndWithError(err)
	}

	return cleanupProductPrefixArchive(absPrefixNameArchiveDir)
}

func cleanupProductPrefixArchive(absPrefixNameArchiveDir string) error {
	cppa := nod.NewProgress(" cleaning up old prefix archives...")
	defer cppa.EndWithResult("done")

	return backups.Cleanup(absPrefixNameArchiveDir, true, cppa)
}
