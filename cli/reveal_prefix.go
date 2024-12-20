package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
	"os"
	"path/filepath"
)

func RevealPrefixHandler(u *url.URL) error {

	name := u.Query().Get("name")

	return RevealPrefix(name)
}

func RevealPrefix(name string) error {

	rpa := nod.Begin("revealing prefix %s...", name)
	defer rpa.EndWithResult("done")

	absPrefixDir, err := data.GetAbsPrefixDir(name)
	if err != nil {
		return rpa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		rpa.EndWithResult("not found")
		return nil
	}

	absPrefixDriveCPath := filepath.Join(absPrefixDir, data.RelPfxDriveCDir)

	if err := currentOsReveal(absPrefixDriveCPath); err != nil {
		return rpa.EndWithError(err)
	}

	return nil

}
