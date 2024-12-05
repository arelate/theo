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

func UpdatePrefixHandler(u *url.URL) error {

	releaseSelector := data.ReleaseSelectorFromUrl(u)

	if releaseSelector == nil {
		releaseSelector = &data.GitHubReleaseSelector{}
	}

	if releaseSelector.Owner == "" && releaseSelector.Repo == "" {
		cos := CurrentOS()
		releaseSelector.Owner = defaultOsWineOwners[cos]
		releaseSelector.Repo = defaultOsWineRepos[cos]
	}

	q := u.Query()

	name := q.Get("name")

	return UpdatePrefix(name, releaseSelector)
}

func UpdatePrefix(name string, releaseSelector *data.GitHubReleaseSelector) error {

	upa := nod.Begin("updating prefix %s...", name)
	defer upa.EndWithResult("done")

	PrintReleaseSelector([]vangogh_local_data.OperatingSystem{CurrentOS()}, releaseSelector)

	prefixesDir, err := pathways.GetAbsRelDir(data.Prefixes)
	if err != nil {
		return upa.EndWithError(err)
	}

	absPrefixDir := filepath.Join(prefixesDir, busan.Sanitize(name))

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		upa.EndWithResult("prefix not initialized")
		return nil
	}

	absWineBin, err := data.GetWineBinary(CurrentOS(), releaseSelector)
	if err != nil {
		return upa.EndWithError(err)
	}

	if _, err := os.Stat(absWineBin); err != nil {
		return upa.EndWithError(err)
	}

	iwpa := nod.Begin(" executing `wineboot --update`, please wait... ")
	defer iwpa.EndWithResult("done")

	if err := data.UpdateWinePrefix(absWineBin, absPrefixDir); err != nil {
		return upa.EndWithError(err)
	}

	return nil
}
