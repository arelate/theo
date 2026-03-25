package cli

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"path/filepath"

	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/coost"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
)

func egsGetClient() (*http.Client, error) {

	cookiesDir := data.Pwd.AbsRelDirPath(data.Cookies, data.Metadata)
	egsCookiePath := filepath.Join(cookiesDir, egsCookiesFilename)

	jar, err := coost.Read(egs_integration.HostUrl(), egsCookiePath)
	if err != nil {
		return nil, err
	}

	client := http.DefaultClient
	client.Jar = jar

	return client, nil
}

func egsGetAccessToken(cookieStr string) (string, error) {

	eggata := nod.Begin("getting EGS access token...")
	defer eggata.Done()

	cookiesDir := data.Pwd.AbsRelDirPath(data.Cookies, data.Metadata)
	egsCookiePath := filepath.Join(cookiesDir, egsCookiesFilename)

	tokensDir := data.Pwd.AbsRelDirPath(data.Tokens, data.Metadata)
	kvTokens, err := kevlar.New(tokensDir, kevlar.JsonExt)
	if err != nil {
		return "", err
	}

	if err = coost.Import(cookieStr, egs_integration.HostUrl(), egsCookiePath); err != nil {
		return "", err
	}

	if kvTokens.Has(egsTokenKey) {
		if err = kvTokens.Cut(egsTokenKey); err != nil {
			return "", err
		}
	}

	var client *http.Client
	client, err = egsGetClient()
	if err != nil {
		return "", err
	}

	var arr *egs_integration.GetApiRedirectResponse
	arr, err = egs_integration.GetApiRedirect(client)
	if err != nil {
		return "", err
	}

	var ptr *egs_integration.PostTokenResponse
	ptr, err = egs_integration.PostToken(arr.AuthorizationCode, egs_integration.GrantTypeAuthorizationCode, client)
	if err != nil {
		return "", err
	}

	if ptr.AccessToken == "" {
		return "", errors.New("failed to get EGS access token")
	}

	return ptr.AccessToken, nil
}

func egsGetStoredToken() (string, error) {

	eggsvta := nod.Begin("getting stored EGS token...")
	defer eggsvta.Done()

	tokensDir := data.Pwd.AbsRelDirPath(data.Tokens, data.Metadata)
	kvTokens, err := kevlar.New(tokensDir, kevlar.JsonExt)
	if err != nil {
		return "", err
	}

	var rcEgsToken io.ReadCloser
	rcEgsToken, err = kvTokens.Get(egsTokenKey)
	if err != nil {
		return "", err
	}

	var gvt egs_integration.GetVerifyTokenResponse
	if err = json.UnmarshalRead(rcEgsToken, &gvt); err != nil {
		return "", err
	}

	return gvt.Token, nil
}

func egsVerifyToken(token string) error {

	gvta := nod.Begin("verifying EGS token...")
	defer gvta.Done()

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	tokensDir := data.Pwd.AbsRelDirPath(data.Tokens, data.Metadata)
	kvTokens, err := kevlar.New(tokensDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if token == "" {
		if token, err = egsGetStoredToken(); err != nil {
			return err
		}
	}

	if token == "" {
		return errors.New("empty access token, re-connect EGS")
	}

	var vtr *egs_integration.GetVerifyTokenResponse
	vtr, err = egs_integration.GetVerifyToken(token, client)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err = json.MarshalWrite(buf, &vtr); err != nil {
		return err
	}

	return kvTokens.Set(egsTokenKey, buf)
}

func egsValidateSupportedPlatform(ii *InstallInfo) error {
	switch ii.OperatingSystem {
	case vangogh_integration.AnyOperatingSystem:
		return errors.New("EGS operations require specific operating system")
	case vangogh_integration.Linux:
		return errors.New("EGS does not support Linux")
	default:
		return nil
	}
}

func egsGetAvailableProducts(ii *InstallInfo) ([]vangogh_integration.AvailableProduct, error) {

	if err := egsValidateSupportedPlatform(ii); err != nil {
		return nil, err
	}

	gameAssets, err := egsReadLocalGameAssets(ii)
	if err != nil {
		return nil, err
	}

	if len(gameAssets) == 0 || ii.force {
		if err = egsFetchGameAssets(ii); err != nil {
			return nil, err
		}

		gameAssets, err = egsReadLocalGameAssets(ii)
		if err != nil {
			return nil, err
		}
	}

	return egsGameAssetsAvailableProducts(gameAssets, ii)
}

func egsGameAssetsAvailableProducts(gameAssets []egs_integration.GameAsset, ii *InstallInfo) ([]vangogh_integration.AvailableProduct, error) {

	if err := egsValidateSupportedPlatform(ii); err != nil {
		return nil, err
	}

	catalogItemsDir := data.Pwd.AbsRelDirPath(data.CatalogItems, data.Metadata)
	kvCatalogItems, err := kevlar.New(catalogItemsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	availableProducts := make([]vangogh_integration.AvailableProduct, 0, len(gameAssets))

	var token string
	var client *http.Client

	for _, gameAsset := range gameAssets {

		if !kvCatalogItems.Has(gameAsset.CatalogItemId) || ii.force {

			if token == "" {
				token, err = egsGetStoredToken()
				if err != nil {
					return nil, err
				}

				if err = egsVerifyToken(token); err != nil {
					return nil, err
				}
			}

			if client == nil {
				client, err = egsGetClient()
				if err != nil {
					return nil, err
				}
			}

			if err = egsFetchCatalogItem(gameAsset.Namespace, gameAsset.CatalogItemId, token, client, kvCatalogItems); err != nil {
				return nil, err
			}

		}

		var catalogItem *egs_integration.CatalogItem
		catalogItem, err = egsReadLocalCatalogItem(gameAsset.CatalogItemId, kvCatalogItems)
		if err != nil {
			return nil, err
		}

		ap := vangogh_integration.AvailableProduct{
			Id:               gameAsset.AppName,
			Title:            catalogItem.Title,
			OperatingSystems: []vangogh_integration.OperatingSystem{ii.OperatingSystem},
		}

		availableProducts = append(availableProducts, ap)
	}

	return availableProducts, nil
}

func egsReadLocalGameAssets(ii *InstallInfo) ([]egs_integration.GameAsset, error) {

	if err := egsValidateSupportedPlatform(ii); err != nil {
		return nil, err
	}

	egsOsApKey := originAvailableProductsKey(ii.Origin, ii.OperatingSystem)

	availableProductsDir := data.Pwd.AbsRelDirPath(data.AvailableProducts, data.Metadata)
	kvAvailableProducts, err := kevlar.New(availableProductsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if !kvAvailableProducts.Has(egsOsApKey) {
		return nil, errors.New("no EGS game assets found for " + ii.OperatingSystem.String())
	}

	rcGameAssets, err := kvAvailableProducts.Get(egsOsApKey)
	if err != nil {
		return nil, err
	}
	defer rcGameAssets.Close()

	var gameAssets []egs_integration.GameAsset
	if err = json.UnmarshalRead(rcGameAssets, &gameAssets); err != nil {
		return nil, err
	}

	return gameAssets, nil
}

func egsFetchGameAssets(ii *InstallInfo) error {

	efapa := nod.Begin(" fetching EGS game assets...")
	defer efapa.Done()

	var err error

	if err = egsValidateSupportedPlatform(ii); err != nil {
		return err
	}

	var client *http.Client
	if client, err = egsGetClient(); err != nil {
		return err
	}

	var token string
	if token, err = egsGetStoredToken(); err != nil {
		return err
	}
	if err = egsVerifyToken(token); err != nil {
		return err
	}

	gameAssets, err := egs_integration.GetGameAssets(egs_integration.Platform(ii.OperatingSystem), token, client)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err = json.MarshalWrite(buf, &gameAssets); err != nil {
		return err
	}

	egsOsApKey := originAvailableProductsKey(ii.Origin, ii.OperatingSystem)

	availableProductsDir := data.Pwd.AbsRelDirPath(data.AvailableProducts, data.Metadata)
	kvAvailableProducts, err := kevlar.New(availableProductsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	return kvAvailableProducts.Set(egsOsApKey, buf)
}

func egsFetchCatalogItems(ii *InstallInfo, gameAssets []egs_integration.GameAsset, token string, client *http.Client, kvCatalogItems kevlar.KeyValues) error {

	efcia := nod.NewProgress(" fetching EGS catalog items...")
	defer efcia.Done()

	if err := egsValidateSupportedPlatform(ii); err != nil {
		return err
	}

	efcia.TotalInt(len(gameAssets))

	for _, gameAsset := range gameAssets {

		if kvCatalogItems.Has(gameAsset.CatalogItemId) && !ii.force {
			efcia.Increment()
			continue
		}

		if err := egsFetchCatalogItem(gameAsset.Namespace, gameAsset.CatalogItemId, token, client, kvCatalogItems); err != nil {
			return err
		}

		efcia.Increment()
	}

	return nil
}

func egsFetchCatalogItem(namespace string, catalogItemId string, token string, client *http.Client, kvCatalogItems kevlar.KeyValues) error {

	efcia := nod.Begin(" fetching catalog item %s...", catalogItemId)
	defer efcia.Done()

	catalogItem, err := egs_integration.GetCatalogItem(namespace, catalogItemId, token, client)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err = json.MarshalWrite(buf, &catalogItem); err != nil {
		return err
	}

	return kvCatalogItems.Set(catalogItemId, buf)
}

func egsReadLocalCatalogItem(catalogItemId string, kvCatalogItems kevlar.KeyValues) (*egs_integration.CatalogItem, error) {

	rcCatalogItem, err := kvCatalogItems.Get(catalogItemId)
	if err != nil {
		return nil, err
	}
	defer rcCatalogItem.Close()

	var catalogItem egs_integration.CatalogItem
	if err = json.UnmarshalRead(rcCatalogItem, &catalogItem); err != nil {
		return nil, err
	}

	return &catalogItem, nil
}

func egsGetGameAsset(appName string, ii *InstallInfo) (*egs_integration.GameAsset, error) {

	egga := nod.Begin("getting EGS game asset...")
	defer egga.Done()

	if err := egsValidateSupportedPlatform(ii); err != nil {
		return nil, err
	}

	availableProductsDir := data.Pwd.AbsRelDirPath(data.AvailableProducts, data.Metadata)
	kvAvailableProducts, err := kevlar.New(availableProductsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	egsOsApKey := originAvailableProductsKey(ii.Origin, ii.OperatingSystem)

	if !kvAvailableProducts.Has(egsOsApKey) || ii.force {
		if err = egsFetchGameAssets(ii); err != nil {
			return nil, err
		}
	}

	gameAssets, err := egsReadLocalGameAssets(ii)
	if err != nil {
		return nil, err
	}

	for _, gameAsset := range gameAssets {
		if gameAsset.AppName == appName {
			return &gameAsset, nil
		}
	}

	return nil, errors.New("game asset not found for appName " + appName)
}
