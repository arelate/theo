package cli

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

	availableProducts := make([]vangogh_integration.AvailableProduct, 0, len(gameAssets))

	for _, gameAsset := range gameAssets {

		catalogItem, err := egsGetCatalogItem(&gameAsset, ii)
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

func egsGetCatalogItem(gameAsset *egs_integration.GameAsset, ii *InstallInfo) (*egs_integration.CatalogItem, error) {

	catalogItemsDir := data.Pwd.AbsRelDirPath(data.CatalogItems, data.Metadata)
	kvCatalogItems, err := kevlar.New(catalogItemsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if !kvCatalogItems.Has(gameAsset.CatalogItemId) || ii.force {

		if err = egsFetchCatalogItem(gameAsset, kvCatalogItems); err != nil {
			return nil, err
		}
	}

	rcCatalogItem, err := kvCatalogItems.Get(gameAsset.CatalogItemId)
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

func egsFetchCatalogItem(gameAsset *egs_integration.GameAsset, kvCatalogItems kevlar.KeyValues) error {

	efcia := nod.Begin(" fetching catalog item %s...", gameAsset.CatalogItemId)
	defer efcia.Done()

	token, err := egsGetStoredToken()
	if err != nil {
		return err
	}

	if err = egsVerifyToken(token); err != nil {
		return err
	}

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	catalogItem, err := egs_integration.GetCatalogItem(gameAsset.Namespace, gameAsset.CatalogItemId, token, client)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err = json.MarshalWrite(buf, &catalogItem); err != nil {
		return err
	}

	return kvCatalogItems.Set(gameAsset.CatalogItemId, buf)
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

func egsGetGameManifest(gameAsset *egs_integration.GameAsset, ii *InstallInfo, force bool) (*egs_integration.GameManifest, error) {

	eggma := nod.Begin("getting EGS game manifest...")
	defer eggma.Done()

	if err := egsValidateSupportedPlatform(ii); err != nil {
		return nil, err
	}

	gameManifestsDir := data.Pwd.AbsRelDirPath(data.GameManifests, data.Metadata)
	kvGameManifests, err := kevlar.New(gameManifestsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	osAppNameKey := fmt.Sprintf("%s-%s", gameAsset.AppName, ii.OperatingSystem)

	if !kvGameManifests.Has(osAppNameKey) || force {
		if err = egsFetchGameManifest(osAppNameKey, gameAsset, ii.OperatingSystem, kvGameManifests); err != nil {
			return nil, err
		}
	}

	rcGameManifest, err := kvGameManifests.Get(osAppNameKey)
	if err != nil {
		return nil, err
	}
	defer rcGameManifest.Close()

	var gameManifest egs_integration.GameManifest
	if err = json.UnmarshalRead(rcGameManifest, &gameManifest); err != nil {
		return nil, err
	}

	return &gameManifest, nil
}

func egsFetchGameManifest(key string, gameAsset *egs_integration.GameAsset, operatingSystem vangogh_integration.OperatingSystem, kvGameManifests kevlar.KeyValues) error {

	efgma := nod.Begin(" fetching game manifest %s...", key)
	defer efgma.Done()

	token, err := egsGetStoredToken()
	if err != nil {
		return err
	}

	if err = egsVerifyToken(token); err != nil {
		return err
	}

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	gameManifest, err := egs_integration.GetGameManifest(
		gameAsset.Namespace,
		gameAsset.CatalogItemId,
		gameAsset.AppName,
		egs_integration.Platform(operatingSystem),
		token, client)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err = json.MarshalWrite(buf, &gameManifest); err != nil {
		return err
	}

	return kvGameManifests.Set(key, buf)
}

func egsGetManifest(appName string, gameManifest *egs_integration.GameManifest, operatingSystem vangogh_integration.OperatingSystem, force bool) (*egs_integration.Manifest, error) {

	egma := nod.Begin("getting EGS manifest...")
	defer egma.Done()

	manifestsDir := data.Pwd.AbsRelDirPath(data.Manifests, data.Metadata)
	kvManifests, err := kevlar.New(manifestsDir, egs_integration.ManifestExt)
	if err != nil {
		return nil, err
	}

	osAppNameKey := fmt.Sprintf("%s-%s", appName, operatingSystem)

	if !kvManifests.Has(osAppNameKey) || force {
		if err = egsFetchManifests(osAppNameKey, gameManifest, kvManifests); err != nil {
			return nil, err
		}
	}

	absManifestFilename := filepath.Join(manifestsDir, osAppNameKey+egs_integration.ManifestExt)

	manifestFile, err := os.Open(absManifestFilename)
	if err != nil {
		return nil, err
	}
	defer manifestFile.Close()

	return egs_integration.ReadBinaryManifest(manifestFile)
}

func egsFetchManifests(key string, gameManifest *egs_integration.GameManifest, kvManifests kevlar.KeyValues) error {

	efma := nod.Begin(" fetching manifests for %s...", key)
	defer efma.Done()

	manifestUrls, err := gameManifest.Urls()
	if err != nil {
		return err
	}

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	var downloaded bool

	for _, manifestUrl := range manifestUrls {
		if err = egsFetchManifest(key, manifestUrl, client, kvManifests); err == nil {
			downloaded = true
			break
		}
	}

	if !downloaded {
		return errors.New("unable to successfully download at least one manifest")
	}

	return nil
}

func egsFetchManifest(key string, manifestUrl *url.URL, client *http.Client, kvManifests kevlar.KeyValues) error {

	req, err := http.NewRequest(http.MethodGet, manifestUrl.String(), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	return kvManifests.Set(key, resp.Body)
}
