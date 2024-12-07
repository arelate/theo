package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
)

func RemovePrefixHandler(u *url.URL) error {

	q := u.Query()
	name := q.Get("name")
	noArchive := q.Has("no-archive")
	force := q.Has("force")

	return RemovePrefix(name, noArchive, force)
}

func RemovePrefix(name string, noArchive, force bool) error {

	rpa := nod.NewProgress("removing prefix %s...", name)
	defer rpa.EndWithResult("done")

	prefixesDir, err := pathways.GetAbsRelDir(data.Prefixes)
	if err != nil {
		return rpa.EndWithError(err)
	}

	absPrefixDir := filepath.Join(prefixesDir, busan.Sanitize(name))

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		rpa.EndWithResult("not present")
		return nil
	}

	if noArchive {
		// do nothing
	} else {
		if err := ArchivePrefix(name); err != nil {
			return rpa.EndWithError(err)
		}
	}

	if !force {
		rpa.EndWithResult("found prefix, use -force to remove")
		return nil
	}

	return os.RemoveAll(absPrefixDir)
}
