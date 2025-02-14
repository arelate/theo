package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/backups"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
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
	defer apa.Done()

	vangogh_integration.PrintParams(ids, nil, []string{langCode}, nil, true)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir, data.SlugProperty)
	if err != nil {
		return err
	}

	apa.TotalInt(len(ids))

	for _, id := range ids {

		if err = archiveProductPrefix(id, langCode, rdx); err != nil {
			return err
		}

		apa.Increment()
	}

	return nil

}

func archiveProductPrefix(id, langCode string, rdx redux.Readable) error {

	appa := nod.Begin(" archiving prefix for %s...", id)
	defer appa.Done()

	prefixArchiveDir, err := pathways.GetAbsRelDir(data.PrefixArchive)
	if err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	absPrefixNameArchiveDir := filepath.Join(prefixArchiveDir, prefixName)

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absPrefixNameArchiveDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absPrefixNameArchiveDir, 0755); err != nil {
			return err
		}
	}

	if err = backups.Compress(absPrefixDir, absPrefixNameArchiveDir); err != nil {
		return err
	}

	return cleanupProductPrefixArchive(absPrefixNameArchiveDir)
}

func cleanupProductPrefixArchive(absPrefixNameArchiveDir string) error {
	cppa := nod.NewProgress(" cleaning up old prefix archives...")
	defer cppa.Done()

	return backups.Cleanup(absPrefixNameArchiveDir, true, cppa)
}
