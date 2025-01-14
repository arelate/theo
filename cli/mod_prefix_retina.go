package cli

import (
	_ "embed"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/cli/pfx_mod"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
)

func ModPrefixRetinaHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	wineRepo := q.Get("wine-repo")
	revert := q.Has("revert")
	force := q.Has("force")

	return ModPrefixRetina(id, langCode, wineRepo, revert, force)
}

func ModPrefixRetina(id, langCode string, wineRepo string, revert, force bool) error {

	mpa := nod.Begin("modding retina in prefix for %s...", id)
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

	wineCtx := &data.WineContext{
		BinPath:    absWineBin,
		PrefixPath: absPrefixDir,
	}

	return pfx_mod.ToggleRetina(wineCtx, revert, force)
}
