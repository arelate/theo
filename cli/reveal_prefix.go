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

const (
	driveCpath = "drive_c"
)

func RevealPrefixHandler(u *url.URL) error {

	name := u.Query().Get("name")

	return RevealPrefix(name)
}

func RevealPrefix(name string) error {

	rpa := nod.Begin("revealing prefix %s...", name)
	defer rpa.EndWithResult("done")

	prefixesDir, err := pathways.GetAbsRelDir(data.Prefixes)
	if err != nil {
		return rpa.EndWithError(err)
	}

	absPrefixDir := filepath.Join(prefixesDir, busan.Sanitize(name))

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		rpa.EndWithResult("not found")
		return nil
	}

	absPrefixDriveCPath := filepath.Join(absPrefixDir, driveCpath)

	if err := revealCurrentOs(absPrefixDriveCPath); err != nil {
		return rpa.EndWithError(err)
	}

	return nil

}
