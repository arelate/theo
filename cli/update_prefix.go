package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
	"os"
)

func UpdatePrefixHandler(u *url.URL) error {

	releaseSelector := data.ReleaseSelectorFromUrl(u)

	q := u.Query()

	name := q.Get("name")

	return UpdatePrefix(name, releaseSelector)
}

func UpdatePrefix(name string, releaseSelector *data.GitHubReleaseSelector) error {

	upa := nod.Begin("updating prefix %s...", name)
	defer upa.EndWithResult("done")

	if releaseSelector == nil {
		releaseSelector = &data.GitHubReleaseSelector{}
	}

	if releaseSelector.Owner == "" && releaseSelector.Repo == "" {
		dws, err := data.GetDefaultWineSource(data.CurrentOS())
		if err != nil {
			return upa.EndWithError(err)
		}
		releaseSelector.Owner = dws.Owner
		releaseSelector.Repo = dws.Repo
	}

	PrintReleaseSelector(releaseSelector)

	absPrefixDir, err := data.GetAbsPrefixDir(name)
	if err != nil {
		return upa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		upa.EndWithResult("prefix not initialized")
		return nil
	}

	absWineBin, err := data.GetWineBinary(data.CurrentOS(), releaseSelector)
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
