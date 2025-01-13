package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/cli/pfx_mod"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

func ModPrefixDxVkHandler(u *url.URL) error {

	q := u.Query()

	name := q.Get("name")

	dxVkRepo := q.Get("dxvk-repo")

	revert := q.Has("revert")

	return ModPrefixDxVk(name, dxVkRepo, revert)
}

func ModPrefixDxVk(name string, dxVkRepo string, revert bool) error {
	mpa := nod.Begin("modding DXVK in prefix %s...", name)
	defer mpa.EndWithResult("done")

	if data.CurrentOS() != vangogh_integration.MacOS {
		mpa.EndWithResult("DXVK prefix mod is only applicable to macOS")
		return nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(name)
	if err != nil {
		return mpa.EndWithError(err)
	}

	dxVkSource, release, err := data.GetDxVkSourceLatestRelease(dxVkRepo)
	if err != nil {
		return mpa.EndWithError(err)
	}

	binariesDir, err := pathways.GetAbsRelDir(data.Binaries)
	if err != nil {
		return mpa.EndWithError(err)
	}

	absBinaryDir := filepath.Join(binariesDir, dxVkSource.Owner, dxVkSource.Repo, busan.Sanitize(release.TagName))

	return pfx_mod.ToggleDxVk(absPrefixDir, absBinaryDir, revert)
}
