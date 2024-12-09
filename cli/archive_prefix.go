package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/backups"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
)

func ArchivePrefixHandler(u *url.URL) error {

	name := u.Query().Get("name")

	return ArchivePrefix(name)
}

func ArchivePrefix(name string) error {

	apa := nod.NewProgress("archiving prefix %s...", name)
	defer apa.EndWithResult("done")

	prefixArchiveDir, err := pathways.GetAbsRelDir(data.PrefixArchive)
	if err != nil {
		return apa.EndWithError(err)
	}

	absPrefixNameArchiveDir := filepath.Join(prefixArchiveDir, busan.Sanitize(name))

	if _, err := os.Stat(absPrefixNameArchiveDir); err != nil {
		if err := os.MkdirAll(absPrefixNameArchiveDir, 0755); err != nil {
			return apa.EndWithError(err)
		}
	}

	absPrefixDir, err := data.GetAbsPrefixDir(name)
	if err != nil {
		return apa.EndWithError(err)
	}

	if err := backups.Compress(absPrefixDir, absPrefixNameArchiveDir); err != nil {
		return apa.EndWithError(err)
	}

	cpa := nod.NewProgress("cleaning up old prefix archives...")
	defer cpa.EndWithResult("done")

	if err := backups.Cleanup(absPrefixNameArchiveDir, true, cpa); err != nil {
		return cpa.EndWithError(err)
	}

	return nil

}
