package cli

import (
	"github.com/arelate/theo/cli/pfx_mod"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

func ModPrefixDxVkHandler(u *url.URL) error {

	q := u.Query()

	name := q.Get("name")

	releaseSelector := data.ReleaseSelectorFromUrl(u)

	revert := q.Has("revert")
	force := q.Has("force")

	return ModPrefixDxVk(name, releaseSelector, revert, force)
}

func ModPrefixDxVk(name string, releaseSelector *data.GitHubReleaseSelector, revert, force bool) error {
	mpa := nod.Begin("modding DXVK in prefix %s...", name)
	defer mpa.EndWithResult("done")

	PrintReleaseSelector(releaseSelector)

	if data.CurrentOS() != vangogh_local_data.MacOS {
		mpa.EndWithResult("DXVK prefix mod is only applicable to macOS")
		return nil
	}

	if releaseSelector == nil {
		releaseSelector = &data.GitHubReleaseSelector{}
	}

	if releaseSelector.Owner == "" && releaseSelector.Repo == "" {
		dxVkSource, err := data.GetFirstDxVkSource(data.CurrentOS())
		if err != nil {
			return mpa.EndWithError(err)
		}
		releaseSelector.Owner = dxVkSource.Owner
		releaseSelector.Repo = dxVkSource.Repo
	}

	absPrefixDir, err := data.GetAbsPrefixDir(name)
	if err != nil {
		return mpa.EndWithError(err)
	}

	dxVkSource, release, err := data.GetDxVkSourceRelease(data.CurrentOS(), releaseSelector)
	if err != nil {
		return mpa.EndWithError(err)
	}

	binariesDir, err := pathways.GetAbsRelDir(data.Binaries)
	if err != nil {
		return mpa.EndWithError(err)
	}

	absBinaryDir := filepath.Join(binariesDir, dxVkSource.Owner, dxVkSource.Repo, busan.Sanitize(release.TagName))

	return pfx_mod.ToggleDxVk(absPrefixDir, absBinaryDir, revert, force)
}
