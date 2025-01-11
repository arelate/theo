package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
	"os"
)

func InitPrefixHandler(u *url.URL) error {

	q := u.Query()

	name := q.Get("name")
	wineRepo := q.Get("wine-repo")
	force := q.Has("force")

	return InitPrefix(name, wineRepo, force)
}

func InitPrefix(name string, wineRepo string, force bool) error {

	cpa := nod.Begin("initializing prefix %s...", name)
	defer cpa.EndWithResult("done")

	absPrefixDir, err := data.GetAbsPrefixDir(name)
	if err != nil {
		return cpa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixDir); err == nil {
		if !force {
			cpa.EndWithResult("already exists")
			return nil
		}
	}

	absWineBin, err := data.GetWineBinary(wineRepo)
	if err != nil {
		return cpa.EndWithError(err)
	}

	if _, err := os.Stat(absWineBin); err != nil {
		return cpa.EndWithError(err)
	}

	iwpa := nod.Begin(" executing `wineboot --init`, please wait... ")
	defer iwpa.EndWithResult("done")

	if err := data.InitWinePrefix(&data.WineContext{
		BinPath:    absWineBin,
		PrefixPath: absPrefixDir,
	}); err != nil {
		return cpa.EndWithError(err)
	}

	return nil
}
