package cli

import (
	_ "embed"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/cli/pfx_mod"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
	"os"
)

func ModPrefixRetinaHandler(u *url.URL) error {

	q := u.Query()

	name := q.Get("name")

	wineRepo := q.Get("wine-repo")

	revert := q.Has("revert")
	force := q.Has("force")

	return ModPrefixRetina(name, wineRepo, revert, force)
}

func ModPrefixRetina(name string, wineRepo string, revert, force bool) error {

	mpa := nod.Begin("modding retina in prefix %s...", name)
	defer mpa.EndWithResult("done")

	if data.CurrentOS() != vangogh_integration.MacOS {
		mpa.EndWithResult("retina prefix mod is only applicable to %s", vangogh_integration.MacOS)
		return nil
	}

	absWineBin, err := data.GetWineBinary(wineRepo)
	if err != nil {
		return mpa.EndWithError(err)
	}

	if _, err := os.Stat(absWineBin); err != nil {
		return mpa.EndWithError(err)
	}

	absPrefixDir, err := data.GetAbsPrefixDir(name)
	if err != nil {
		return mpa.EndWithError(err)
	}

	wineCtx := &data.WineContext{
		BinPath:    absWineBin,
		PrefixPath: absPrefixDir,
	}

	return pfx_mod.ToggleRetina(wineCtx, revert, force)
}
