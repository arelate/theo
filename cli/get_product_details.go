package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

func getProductDetails(id string, rdx redux.Writeable, force bool) (*vangogh_integration.ProductDetails, error) {

	gpda := nod.NewProgress(" getting product details for %s...", id)
	defer gpda.Done()

	productDetailsDir, err := pathways.GetAbsRelDir(data.ProductDetails)
	if err != nil {
		return nil, err
	}

	kvProductDetails, err := kevlar.New(productDetailsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if dm, err := readLocalProductDetails(id, kvProductDetails); err != nil {
		return nil, err
	} else if dm != nil && !force {
		gpda.EndWithResult("read local")
		return dm, nil
	}

	if err = validateSessionToken(rdx); err != nil {
		return nil, err
	}

	productDetails, err := fetchRemoteProductDetails(id, rdx, kvProductDetails)
	if err != nil {
		return nil, err
	}

	gpda.EndWithResult("fetched remote")

	if err = reduceProductDetails(id, productDetails, rdx); err != nil {
		return nil, err
	}

	return productDetails, nil
}

func readLocalProductDetails(id string, kvProductDetails kevlar.KeyValues) (*vangogh_integration.ProductDetails, error) {

	if has := kvProductDetails.Has(id); !has {
		return nil, nil
	}

	tmReadCloser, err := kvProductDetails.Get(id)
	if err != nil {
		return nil, err
	}
	defer tmReadCloser.Close()

	var productDetails vangogh_integration.ProductDetails
	if err = json.NewDecoder(tmReadCloser).Decode(&productDetails); err != nil {
		return nil, err
	}

	return &productDetails, nil
}

func fetchRemoteProductDetails(id string, rdx redux.Readable, kvProductDetails kevlar.KeyValues) (*vangogh_integration.ProductDetails, error) {

	fra := nod.Begin(" fetching remote product details for %s...", id)
	defer fra.Done()

	query := url.Values{
		vangogh_integration.IdProperty: {id},
	}

	req, err := data.ServerRequest(http.MethodGet, data.ApiProductDetailsPath, query, rdx)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New("error fetching product details: " + resp.Status)
	}

	var bts []byte
	buf := bytes.NewBuffer(bts)
	tr := io.TeeReader(resp.Body, buf)

	if err = kvProductDetails.Set(id, tr); err != nil {
		return nil, err
	}

	var productDetails vangogh_integration.ProductDetails
	if err = json.NewDecoder(buf).Decode(&productDetails); err != nil {
		return nil, err
	}

	return &productDetails, nil
}

func reduceProductDetails(id string, productDetails *vangogh_integration.ProductDetails, rdx redux.Writeable) error {

	rpda := nod.Begin(" reducing product details...")
	defer rpda.Done()

	propertyValues := make(map[string][]string)

	oss := make([]string, 0, len(productDetails.OperatingSystems))
	for _, os := range productDetails.OperatingSystems {
		oss = append(oss, os.String())
	}

	propertyValues[vangogh_integration.SlugProperty] = []string{productDetails.Slug}
	propertyValues[vangogh_integration.SteamAppIdProperty] = []string{productDetails.SteamAppId}
	propertyValues[vangogh_integration.TitleProperty] = []string{productDetails.Title}
	propertyValues[vangogh_integration.OperatingSystemsProperty] = oss
	propertyValues[vangogh_integration.DevelopersProperty] = productDetails.Developers
	propertyValues[vangogh_integration.PublishersProperty] = productDetails.Publishers
	propertyValues[vangogh_integration.VerticalImageProperty] = []string{productDetails.Images.VerticalImage}
	propertyValues[vangogh_integration.ImageProperty] = []string{productDetails.Images.Image}
	propertyValues[vangogh_integration.HeroProperty] = []string{productDetails.Images.Hero}
	propertyValues[vangogh_integration.LogoProperty] = []string{productDetails.Images.Logo}
	propertyValues[vangogh_integration.IconProperty] = []string{productDetails.Images.Icon}
	propertyValues[vangogh_integration.IconSquareProperty] = []string{productDetails.Images.IconSquare}
	propertyValues[vangogh_integration.BackgroundProperty] = []string{productDetails.Images.Background}

	for property, values := range propertyValues {
		if err := rdx.ReplaceValues(property, id, values...); err != nil {
			return err
		}
	}

	return nil
}
