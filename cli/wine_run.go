package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

func WineRunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	wineRepo := q.Get("wine-repo")
	exePath := q.Get("exe-path")

	return WineRun(id, langCode, wineRepo, exePath)
}

func WineRun(id string, langCode string, wineRepo, exePath string) error {

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SlugProperty)
	if err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, langCode, rdx)
	if err != nil {
		return err
	}

	absPrefixPath, err := data.GetAbsPrefixDir(prefixName)
	if err != nil {
		return err
	}

	absExePath := filepath.Join(absPrefixPath, data.RelPrefixDriveCDir, exePath)

	absWineBin, err := data.GetWineBinary(wineRepo)
	if err != nil {
		return err
	}

	wcx := &data.WineContext{
		BinPath:    absWineBin,
		PrefixPath: absPrefixPath,
	}

	if exePath != "" {
		return data.RunWineExePath(wcx, absExePath)
	} else {
		return data.RunWineDefaultGogLnk(wcx)
	}
}
