package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path/filepath"
)

const relWineCfgPath = "windows/system32/winecfg.exe"

func WineCfgHandler(u *url.URL) error {
	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	verbose := q.Has("verbose")
	force := q.Has("force")

	return WineCfg(id, langCode, verbose, force)
}

func WineCfg(id, langCode string, verbose, force bool) error {

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	absWineCfgPath := filepath.Join(absPrefixDir, relPrefixDriveCDir, relWineCfgPath)

	et := &execTask{
		exe:     absWineCfgPath,
		verbose: verbose,
	}

	return WineRun(id, langCode, et, force)
}
