package cli

import (
	_ "embed"
	"github.com/arelate/theo/cli/pfx_mod"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"net/url"
	"os"
)

func ModPrefixRetinaHandler(u *url.URL) error {

	q := u.Query()

	name := q.Get("name")

	releaseSelector := data.ReleaseSelectorFromUrl(u)

	revert := q.Has("revert")
	force := q.Has("force")

	return ModPrefixRetina(name, releaseSelector, revert, force)
}

func ModPrefixRetina(name string, releaseSelector *data.GitHubReleaseSelector, revert, force bool) error {

	mpa := nod.Begin("modding retina in prefix %s...", name)
	defer mpa.EndWithResult("done")

	PrintReleaseSelector([]vangogh_local_data.OperatingSystem{CurrentOS()}, releaseSelector)

	if CurrentOS() != vangogh_local_data.MacOS {
		mpa.EndWithResult("retina prefix mod is only applicable to macOS")
		return nil
	}

	if releaseSelector == nil {
		releaseSelector = &data.GitHubReleaseSelector{}
	}

	if releaseSelector.Owner == "" && releaseSelector.Repo == "" {
		dws, err := data.GetDefaultWineSource(CurrentOS())
		if err != nil {
			return mpa.EndWithError(err)
		}
		releaseSelector.Owner = dws.Owner
		releaseSelector.Repo = dws.Repo
	}

	absWineBin, err := data.GetWineBinary(CurrentOS(), releaseSelector)
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
