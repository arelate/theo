package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
	"os"
)

func UpdatePrefixHandler(u *url.URL) error {

	q := u.Query()

	wineRepo := q.Get("wine-repo")
	name := q.Get("name")

	return UpdatePrefix(name, wineRepo)
}

func UpdatePrefix(name string, wineRepo string) error {

	upa := nod.Begin("updating prefix %s...", name)
	defer upa.EndWithResult("done")

	absPrefixDir, err := data.GetAbsPrefixDir(name)
	if err != nil {
		return upa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		upa.EndWithResult("prefix not initialized")
		return nil
	}

	absWineBin, err := data.GetWineBinary(wineRepo)
	if err != nil {
		return upa.EndWithError(err)
	}

	if _, err := os.Stat(absWineBin); err != nil {
		return upa.EndWithError(err)
	}

	iwpa := nod.Begin(" executing `wineboot --update`, please wait... ")
	defer iwpa.EndWithResult("done")

	if err := data.UpdateWinePrefix(&data.WineContext{
		BinPath:    absWineBin,
		PrefixPath: absPrefixDir,
	}); err != nil {
		return upa.EndWithError(err)
	}

	return nil
}
