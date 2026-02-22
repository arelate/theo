package data

import (
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
)

type OriginData struct {
	ProductDetails   *vangogh_integration.ProductDetails
	AppInfoKv        steam_vdf.ValveDataFile
	OperatingSystems []vangogh_integration.OperatingSystem
}
