package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"strings"
)

func UpdatePrefixHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	wineRepo := q.Get("wine-repo")

	return UpdatePrefix(langCode, wineRepo, ids...)
}

func UpdatePrefix(langCode string, wineRepo string, ids ...string) error {

	upa := nod.NewProgress("updating prefixes for %s...", strings.Join(ids, ","))
	defer upa.EndWithResult("done")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return upa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SlugProperty)
	if err != nil {
		return upa.EndWithError(err)
	}

	absWineBin, err := data.GetWineBinary(wineRepo)
	if err != nil {
		return upa.EndWithError(err)
	}

	if _, err := os.Stat(absWineBin); err != nil {
		return upa.EndWithError(err)
	}

	upa.TotalInt(len(ids))

	for _, id := range ids {
		if err := updateProductPrefix(id, langCode, rdx, absWineBin); err != nil {
			return upa.EndWithError(err)
		}
		upa.Increment()
	}

	return nil
}

func updateProductPrefix(id, langCode string, rdx kevlar.ReadableRedux, absWineBin string) error {
	uppa := nod.Begin(" updating prefix for %s...", id)
	defer uppa.EndWithResult("done")

	prefixName, err := data.GetPrefixName(id, langCode, rdx)
	if err != nil {
		return uppa.EndWithError(err)
	}

	if prefixName == "" {
		uppa.EndWithResult("prefix for %s was not created", id)
		return nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(prefixName)
	if err != nil {
		return uppa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixDir); os.IsNotExist(err) {
		uppa.EndWithResult("not present")
		return nil
	}

	return data.UpdateWinePrefix(&data.WineContext{
		BinPath:    absWineBin,
		PrefixPath: absPrefixDir,
	})
}
