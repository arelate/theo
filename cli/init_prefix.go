package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"os"
	"strings"
)

func initPrefix(langCode, wineRepo string, force bool, ids ...string) error {

	cpa := nod.NewProgress("initializing prefixes for %s...", strings.Join(ids, ","))
	defer cpa.EndWithResult("done")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return cpa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SlugProperty)
	if err != nil {
		return cpa.EndWithError(err)
	}

	absWineBin, err := data.GetWineBinary(wineRepo)
	if err != nil {
		return cpa.EndWithError(err)
	}

	if _, err := os.Stat(absWineBin); err != nil {
		return cpa.EndWithError(err)
	}

	cpa.TotalInt(len(ids))

	for _, id := range ids {
		if err := initProductPrefix(id, langCode, rdx, absWineBin, force); err != nil {
			return cpa.EndWithError(err)
		}
		cpa.Increment()
	}

	return nil
}

func initProductPrefix(id, langCode string, rdx kevlar.ReadableRedux, absWineBin string, force bool) error {
	ippa := nod.Begin(" initializing prefix for %s...", id)
	defer ippa.EndWithResult("done")

	prefixName, err := data.GetPrefixName(id, langCode, rdx)
	if err != nil {
		return ippa.EndWithError(err)
	}

	absPrefixDir, err := data.GetAbsPrefixDir(prefixName)
	if err != nil {
		return ippa.EndWithError(err)
	}

	if _, err := os.Stat(absPrefixDir); err == nil && !force {
		ippa.EndWithResult("already exists")
		return nil
	}

	return data.InitWinePrefix(&data.WineContext{
		BinPath:    absWineBin,
		PrefixPath: absPrefixDir,
	})
}
