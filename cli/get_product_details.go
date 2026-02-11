package cli

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func getProductDetails(id string, rdx redux.Writeable, force bool) (*vangogh_integration.ProductDetails, error) {

	gpda := nod.NewProgress(" getting product details for %s...", id)
	defer gpda.Done()

	productDetailsDir := data.Pwd.AbsRelDirPath(data.ProductDetails, vangogh_integration.Metadata)

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

	if err = vangoghValidateSessionToken(rdx); err != nil {
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
	if err = json.UnmarshalRead(tmReadCloser, &productDetails); err != nil {
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

	req, err := data.VangoghRequest(http.MethodGet, data.ApiProductDetailsPath, query, rdx)
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
	if err = json.UnmarshalRead(buf, &productDetails); err != nil {
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

	reductionProperties := []string{
		vangogh_integration.SlugProperty,
		vangogh_integration.SteamAppIdProperty,
		vangogh_integration.TitleProperty,
		vangogh_integration.OperatingSystemsProperty,
		vangogh_integration.DevelopersProperty,
		vangogh_integration.PublishersProperty,
		vangogh_integration.VerticalImageProperty,
		vangogh_integration.ImageProperty,
		vangogh_integration.HeroProperty,
		vangogh_integration.LogoProperty,
		vangogh_integration.IconProperty,
		vangogh_integration.IconSquareProperty,
		vangogh_integration.BackgroundProperty,
	}

	for _, property := range reductionProperties {

		var values []string

		switch property {
		case vangogh_integration.SlugProperty:
			values = []string{productDetails.Slug}
		case vangogh_integration.SteamAppIdProperty:
			values = []string{productDetails.SteamAppId}
		case vangogh_integration.TitleProperty:
			values = []string{productDetails.Title}
		case vangogh_integration.OperatingSystemsProperty:
			values = oss
		case vangogh_integration.DevelopersProperty:
			values = productDetails.Developers
		case vangogh_integration.PublishersProperty:
			values = productDetails.Publishers
		case vangogh_integration.VerticalImageProperty:
			values = []string{productDetails.Images.VerticalImage}
		case vangogh_integration.ImageProperty:
			values = []string{productDetails.Images.Image}
		case vangogh_integration.HeroProperty:
			values = []string{productDetails.Images.Hero}
		case vangogh_integration.LogoProperty:
			values = []string{productDetails.Images.Logo}
		case vangogh_integration.IconProperty:
			values = []string{productDetails.Images.Icon}
		case vangogh_integration.IconSquareProperty:
			values = []string{productDetails.Images.IconSquare}
		case vangogh_integration.BackgroundProperty:
			values = []string{productDetails.Images.Background}
		}

		if len(values) == 1 && values[0] == "" {
			values = nil
		}

		if len(values) > 0 {
			propertyValues[property] = values
		}
	}

	for property, values := range propertyValues {
		if err := rdx.ReplaceValues(property, id, values...); err != nil {
			return err
		}
	}

	return nil
}
