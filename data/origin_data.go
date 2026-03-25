package data

import (
	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
)

type OriginData struct {
	ProductDetails *vangogh_integration.ProductDetails
	AppInfoKv      steam_vdf.ValveDataFile
	//GameAsset      *egs_integration.GameAsset
	CatalogItem  *egs_integration.CatalogItem
	GameManifest *egs_integration.GameManifest
	Manifest     *egs_integration.Manifest
}
