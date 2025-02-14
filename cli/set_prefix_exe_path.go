package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func SetPrefixExePathHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	exePath := q.Get("exe-path")

	return SetPrefixExePath(langCode, exePath, ids...)
}

func SetPrefixExePath(langCode string, exePath string, ids ...string) error {

	spepa := nod.NewProgress("setting prefix exe path for wine-run...")
	defer spepa.Done()

	if strings.HasPrefix(exePath, ".") ||
		strings.HasPrefix(exePath, "/") {
		spepa.EndWithResult("exe path must be relative and cannot start with . or /")
		return nil
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.PrefixExePathProperty)
	if err != nil {
		return err
	}

	exePaths := make(map[string][]string)

	spepa.TotalInt(len(ids))

	for _, id := range ids {

		var absPrefixDir string
		absPrefixDir, err = data.GetAbsPrefixDir(id, langCode, rdx)
		if err != nil {
			return err
		}

		absExePath := filepath.Join(absPrefixDir, relPrefixDriveCDir, exePath)
		if _, err = os.Stat(absExePath); err != nil {
			spepa.Error(err)
			spepa.Increment()
			continue
		}

		prefixName, err := data.GetPrefixName(id, rdx)
		if err != nil {
			return err
		}

		exePaths[path.Join(prefixName, langCode)] = []string{exePath}

		spepa.Increment()
	}

	if err := rdx.BatchReplaceValues(data.PrefixExePathProperty, exePaths); err != nil {
		return err
	}

	return nil
}
