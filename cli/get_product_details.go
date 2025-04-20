package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"net/http"
)

func getProductDetails(id string, rdx redux.Writeable, force bool) (*vangogh_integration.ProductDetails, error) {

	gpda := nod.NewProgress(" getting product details...")
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

	// TODO: replace with reduction func to extract more properties

	if err = rdx.MustHave(data.ServerConnectionProperties, vangogh_integration.TitleProperty, vangogh_integration.SlugProperty); err != nil {
		return nil, err
	}

	defer gpda.EndWithResult("fetched remote")
	if dm, err := fetchRemoteProductDetails(id, rdx, kvProductDetails); err != nil {
		return nil, err
	} else {

		if err = rdx.ReplaceValues(vangogh_integration.TitleProperty, id, dm.Title); err != nil {
			return nil, err
		}
		if err = rdx.ReplaceValues(vangogh_integration.SlugProperty, id, dm.Slug); err != nil {
			return nil, err
		}

		return dm, nil
	}
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

	vdmu, err := data.ServerUrl(rdx,
		data.ApiProductDetailsPath,
		map[string]string{vangogh_integration.IdProperty: id})
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Get(vdmu.String())
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

	if err := kvProductDetails.Set(id, tr); err != nil {
		return nil, err
	}

	var productDetails vangogh_integration.ProductDetails
	if err = json.NewDecoder(buf).Decode(&productDetails); err != nil {
		return nil, err
	}

	return &productDetails, nil
}
