package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
	"maps"
	"slices"
)

type ProductsDetails []*vangogh_integration.ProductDetails

func (psd ProductsDetails) Filter(productTypes ...string) ProductsDetails {
	filtered := make(ProductsDetails, 0, len(psd))

	for _, pd := range psd {
		if !slices.Contains(productTypes, pd.ProductType) {
			continue
		}
		filtered = append(filtered, pd)
	}

	return filtered
}

func (psd ProductsDetails) Ids() []string {
	ids := make([]string, 0, len(psd))

	for _, pd := range psd {
		ids = append(ids, pd.Id)
	}

	return ids
}

func (psd ProductsDetails) GameAndIncludedGamesIds() []string {
	ids := make(map[string]any)

	for _, pd := range psd {
		switch pd.ProductType {
		case vangogh_integration.PackProductType:
			for _, includedId := range pd.IncludesGames {
				ids[includedId] = nil
			}
		case vangogh_integration.GameProductType:
			ids[pd.Id] = nil
		case vangogh_integration.DlcProductType:
			// do nothing
		}
	}

	return slices.Collect(maps.Keys(ids))
}

func GetProductsDetails(rdx redux.Writeable, force bool, ids ...string) (ProductsDetails, error) {

	gpda := nod.NewProgress("getting multiple products details...")
	gpda.Done()

	gpda.TotalInt(len(ids))

	psd := make(ProductsDetails, 0, len(ids))

	for _, id := range ids {

		pd, err := GetProductDetails(id, rdx, force)
		if err != nil {
			return nil, err
		}

		if pd.ProductType == "" {
			return nil, errors.New("product details are missing product type")
		}

		psd = append(psd, pd)
		gpda.Increment()
	}

	return psd, nil
}
