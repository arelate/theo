package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/cli/pfx_mod"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

func ModPrefixDxVkHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	dxVkRepo := q.Get("dxvk-repo")
	revert := q.Has("revert")

	return ModPrefixDxVk(id, langCode, dxVkRepo, revert)
}

func ModPrefixDxVk(id, langCode string, dxVkRepo string, revert bool) error {
	mpa := nod.Begin("modding DXVK in prefix for %s...", id)
	defer mpa.EndWithResult("done")

	if data.CurrentOS() != vangogh_integration.MacOS {
		mpa.EndWithResult("DXVK prefix mod is only applicable to macOS")
		return nil
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return mpa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SlugProperty)
	if err != nil {
		return mpa.EndWithError(err)
	}

	prefixName, err := data.GetPrefixName(id, langCode, rdx)
	if err != nil {
		return mpa.EndWithError(err)
	}

	if prefixName == "" {
		mpa.EndWithResult("prefix for %s was not created", id)
		return nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(prefixName)
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
