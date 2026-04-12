package cli

import (
	"encoding/json/v2"
	"net/http"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
)

func gogGetGamesDbEpic(appName string, force bool) (*gog_integration.GamesDbProduct, error) {

	ggegda := nod.Begin("getting GamesDB EGS product...")
	defer ggegda.Done()

	gamesDbDir := data.Pwd.AbsRelDirPath(data.GamesDB, data.Metadata)
	kvGamesDb, err := kevlar.New(gamesDbDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if !kvGamesDb.Has(appName) || force {
		if err = gogFetchGamesDbEpic(appName, kvGamesDb); err != nil {
			return nil, err
		}
	}

	rcGamesDbProduct, err := kvGamesDb.Get(appName)
	if err != nil {
		return nil, err
	}
	defer rcGamesDbProduct.Close()

	var gamesDbProduct gog_integration.GamesDbProduct
	if err = json.UnmarshalRead(rcGamesDbProduct, &gamesDbProduct); err != nil {
		return nil, err
	}

	return &gamesDbProduct, nil
}

func gogFetchGamesDbEpic(appName string, kvGamesDb kevlar.KeyValues) error {

	gfgbea := nod.Begin(" fetching GamesDB EGS product...")
	defer gfgbea.Done()

	gamesDbEpicUrl := gog_integration.GamesDbEpicUrl(appName)
	resp, err := http.DefaultClient.Get(gamesDbEpicUrl.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return kvGamesDb.Set(appName, resp.Body)
}
