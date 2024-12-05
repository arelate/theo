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

var defaultOsWineOwners = map[vangogh_local_data.OperatingSystem]string{
	vangogh_local_data.MacOS: "Gcenx",
	vangogh_local_data.Linux: "GloriousEggroll",
}

var defaultOsWineRepos = map[vangogh_local_data.OperatingSystem]string{
	vangogh_local_data.MacOS: "game-porting-toolkit",
	vangogh_local_data.Linux: "proton-ge-custom",
}

func InitPrefixHandler(u *url.URL) error {

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
	force := q.Has("force")

	return InitPrefix(name, releaseSelector, force)
}

func InitPrefix(name string, releaseSelector *data.GitHubReleaseSelector, force bool) error {

	cpa := nod.Begin("initializing prefix %s...", name)
	defer cpa.EndWithResult("done")

	PrintReleaseSelector([]vangogh_local_data.OperatingSystem{CurrentOS()}, releaseSelector)

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

	iwpa := nod.Begin(" wineboot --init is running, please wait... ")
	defer iwpa.EndWithResult("done")

	if err := data.InitWinePrefix(absWineBin, absPrefixDir); err != nil {
		return cpa.EndWithError(err)
	}

	return nil
}
