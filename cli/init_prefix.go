package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
)

func InitPrefixHandler(u *url.URL) error {

	releaseSelector := data.ReleaseSelectorFromUrl(u)

	q := u.Query()

	name := q.Get("name")
	force := q.Has("force")

	return InitPrefix(name, releaseSelector, force)
}

func InitPrefix(name string, releaseSelector *data.GitHubReleaseSelector, force bool) error {

	cpa := nod.Begin("initializing prefix %s...", name)
	defer cpa.EndWithResult("done")

	PrintReleaseSelector([]vangogh_local_data.OperatingSystem{CurrentOS()}, releaseSelector)

	if releaseSelector == nil {
		releaseSelector = &data.GitHubReleaseSelector{}
	}

	if releaseSelector.Owner == "" && releaseSelector.Repo == "" {
		dws, err := data.GetDefaultWineSource(CurrentOS())
		if err != nil {
			return cpa.EndWithError(err)
		}
		releaseSelector.Owner = dws.Owner
		releaseSelector.Repo = dws.Repo
	}

	prefixesDir, err := pathways.GetAbsRelDir(data.Prefixes)
	if err != nil {
		return cpa.EndWithError(err)
	}

	absPrefixDir := filepath.Join(prefixesDir, busan.Sanitize(name))

	if _, err := os.Stat(absPrefixDir); err == nil {
		if !force {
			cpa.EndWithResult("already exists")
			return nil
		}
	}

	absWineBin, err := data.GetWineBinary(CurrentOS(), releaseSelector)
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
