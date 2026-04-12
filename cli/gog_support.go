package cli

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/url"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
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

func gogShortcutAssets(productDetails *vangogh_integration.ProductDetails, rdx redux.Readable) (map[steam_grid.Asset]*url.URL, error) {

	shortcutAssets := make(map[steam_grid.Asset]*url.URL)

	for _, asset := range steam_grid.ShortcutAssets {

		var imageId string
		var imageType vangogh_integration.ImageType

		switch asset {
		case steam_grid.Header:
			imageId = productDetails.Images.Image
			imageType = vangogh_integration.Image
		case steam_grid.LibraryCapsule:
			imageId = productDetails.Images.VerticalImage
			imageType = vangogh_integration.VerticalImage
		case steam_grid.LibraryHero:
			if productDetails.Images.Hero != "" {
				imageId = productDetails.Images.Hero
				imageType = vangogh_integration.Hero
			} else {
				imageId = productDetails.Images.Background
				imageType = vangogh_integration.Background
			}
		case steam_grid.LibraryLogo:
			imageId = productDetails.Images.Logo
			imageType = vangogh_integration.Logo
		case steam_grid.ClientIcon:
			if productDetails.Images.IconSquare != "" {
				imageId = productDetails.Images.IconSquare
				imageType = vangogh_integration.IconSquare
			} else {
				imageId = productDetails.Images.Icon
				imageType = vangogh_integration.Icon
			}
		default:
			return nil, errors.New("unexpected shortcut asset " + asset.String())
		}

		if imageId != "" {

			imageExt, err := vangogh_integration.ImagePropertyExt(imageType)
			if err != nil {
				return nil, err
			}

			gogImageUrl, err := gog_integration.ImageUrl(imageId, imageExt)
			if err != nil {
				return nil, err
			}

			shortcutAssets[asset] = gogImageUrl
		}
	}

	return shortcutAssets, nil

}
