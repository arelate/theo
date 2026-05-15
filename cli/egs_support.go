package cli

import (
	"bytes"
	"crypto/sha1"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/coost"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	egsCookiesFilename = "egs-cookies.json"
	egsTokenKey        = "egs-token"
	jsonCatalogItemPfx = "{\"id\""
)

var egsClient *http.Client
var egsTokenVerifiedRecently bool

var (
	eosOverlayGameAsset = egs_integration.GameAsset{
		AppName:       "98bc04bc842e4906993fd6d6644ffb8d",
		LabelName:     "Epic Online Services Overlay",
		CatalogItemId: "cc15684f44d849e89e9bf4cec0508b68",
		Namespace:     "302e5ede476149b1bc3e4fe6ae45e50e",
	}

	eosHelperGameAsset = egs_integration.GameAsset{
		AppName:       "c9e2eb9993a1496c99dc529b49a07339",
		LabelName:     "Epic Online Services Helper",
		Namespace:     "302e5ede476149b1bc3e4fe6ae45e50e",
		CatalogItemId: "1108a9c0af47438da91331753b22ea21",
	}
)

func egsGetClient() (*http.Client, error) {

	if egsClient == nil {
		cookiesDir := data.Pwd.AbsRelDirPath(data.Cookies, data.Metadata)
		egsCookiePath := filepath.Join(cookiesDir, egsCookiesFilename)

		jar, err := coost.Read(egs_integration.HostUrl(), egsCookiePath)
		if err != nil {
			return nil, err
		}

		egsClient = http.DefaultClient
		egsClient.Jar = jar
	}

	return egsClient, nil
}

func egsGetAccessToken(cookieStr string) error {

	eggata := nod.Begin("getting EGS access token...")
	defer eggata.Done()

	cookiesDir := data.Pwd.AbsRelDirPath(data.Cookies, data.Metadata)
	egsCookiePath := filepath.Join(cookiesDir, egsCookiesFilename)

	tokensDir := data.Pwd.AbsRelDirPath(data.Tokens, data.Metadata)
	kvTokens, err := kevlar.New(tokensDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if err = coost.Import(cookieStr, egs_integration.HostUrl(), egsCookiePath); err != nil {
		return err
	}

	if kvTokens.Has(egsTokenKey) {
		if err = kvTokens.Cut(egsTokenKey); err != nil {
			return err
		}
	}

	var client *http.Client
	client, err = egsGetClient()
	if err != nil {
		return err
	}

	var apiRedirectResponse egs_integration.GetApiRedirectResponse
	var rcApiRedirectResponse io.ReadCloser

	rcApiRedirectResponse, err = egs_integration.GetApiRedirect(client)
	if err != nil {
		return err
	}

	defer rcApiRedirectResponse.Close()

	if err = json.UnmarshalRead(rcApiRedirectResponse, &apiRedirectResponse); err != nil {
		return err
	}

	return egsPostToken(apiRedirectResponse.AuthorizationCode, egs_integration.GrantTypeAuthorizationCode)
}

func egsRefreshToken(refreshToken string) error {
	erta := nod.Begin("refreshing EGS token...")
	defer erta.Done()

	if refreshToken == "" {
		return errors.New("refresh token not present")
	}

	return egsPostToken(refreshToken, egs_integration.GrantTypeRefreshToken)
}

func egsPostToken(token string, grantType egs_integration.GrantType) error {

	var err error

	var client *http.Client
	client, err = egsGetClient()
	if err != nil {
		return err
	}

	tokensDir := data.Pwd.AbsRelDirPath(data.Tokens, data.Metadata)
	kvTokens, err := kevlar.New(tokensDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	var rcPostTokenResponse io.ReadCloser

	rcPostTokenResponse, err = egs_integration.PostToken(token, grantType, client)
	if err != nil {
		return err
	}

	defer rcPostTokenResponse.Close()

	return kvTokens.Set(egsTokenKey, rcPostTokenResponse)
}

func egsGetStoredPostTokenResponse() (*egs_integration.PostTokenResponse, error) {
	egsptr := nod.Begin("getting stored EGS post token response...")
	defer egsptr.Done()

	tokensDir := data.Pwd.AbsRelDirPath(data.Tokens, data.Metadata)
	kvTokens, err := kevlar.New(tokensDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	var rcEgsToken io.ReadCloser
	rcEgsToken, err = kvTokens.Get(egsTokenKey)
	if err != nil {
		return nil, err
	}
	defer rcEgsToken.Close()

	var ptr egs_integration.PostTokenResponse
	if err = json.UnmarshalRead(rcEgsToken, &ptr); err != nil {
		return nil, err
	}

	return &ptr, nil
}

func egsVerifyToken(client *http.Client) (*egs_integration.PostTokenResponse, error) {

	gvta := nod.Begin("verifying EGS token...")
	defer gvta.Done()

	ptr, err := egsGetStoredPostTokenResponse()
	if err != nil {
		return nil, err
	}

	if ptr.AccessToken == "" {
		return nil, errors.New("empty access token, re-connect EGS")
	}

	if egsTokenVerifiedRecently {
		gvta.EndWithResult("verified recently")
		return ptr, nil
	}

	if ptr.ExpiresAt.Sub(time.Now()) < time.Hour {
		if err = egsRefreshToken(ptr.RefreshToken); err != nil {
			return nil, err
		}

		if ptr, err = egsGetStoredPostTokenResponse(); err != nil {
			return nil, err
		}
	}

	var rcVerifyTokenResponse io.ReadCloser

	rcVerifyTokenResponse, err = egs_integration.GetVerifyToken(ptr.AccessToken, client)
	if err != nil {
		return nil, err
	}

	defer rcVerifyTokenResponse.Close()

	var verifyTokenResponse egs_integration.GetVerifyTokenResponse

	if err = json.UnmarshalRead(rcVerifyTokenResponse, &verifyTokenResponse); err != nil {
		return nil, err
	}

	if verifyTokenResponse.Token == "" {
		return nil, errors.New("empty access token, re-connect EGS")
	}

	if ptr.ExpiresAt.Sub(time.Now()) > time.Hour*3 {
		egsTokenVerifiedRecently = true
	}

	return ptr, nil
}

func egsGameAssetOperatingSystems(appName string, force bool) ([]vangogh_integration.OperatingSystem, error) {

	osGameAssets, err := egsGetGameAssets(force)
	if err != nil {
		return nil, err
	}

	operatingSystems := make([]vangogh_integration.OperatingSystem, 0)

	for sos, gameAssets := range osGameAssets {
		for _, gameAsset := range gameAssets {
			if gameAsset.AppName == appName {
				operatingSystems = append(operatingSystems, sos)
			}
		}
	}

	return operatingSystems, nil
}

func egsGetGameAssets(update bool) (map[vangogh_integration.OperatingSystem][]egs_integration.GameAsset, error) {

	osGameAssets := make(map[vangogh_integration.OperatingSystem][]egs_integration.GameAsset)

	for _, sos := range egs_integration.SupportedOperatingSystems {
		gameAssets, err := egsReadLocalGameAssets(sos)
		if err != nil {
			return nil, err
		}

		if len(gameAssets) == 0 || update {
			if err = egsFetchGameAssets(sos); err != nil {
				return nil, err
			}

			gameAssets, err = egsReadLocalGameAssets(sos)
			if err != nil {
				return nil, err
			}
		}

		osGameAssets[sos] = gameAssets
	}

	return osGameAssets, nil
}

func availableProductIndex(appName string, availableProducts []vangogh_integration.AvailableProduct) int {
	for ii, ap := range availableProducts {
		if ap.Id == appName {
			return ii
		}
	}
	return -1
}

func egsGameAssetsAvailableProducts(
	osGameAssets map[vangogh_integration.OperatingSystem][]egs_integration.GameAsset,
	ii *InstallInfo,
	rdx redux.Writeable) ([]vangogh_integration.AvailableProduct, error) {

	availableProducts := make([]vangogh_integration.AvailableProduct, 0)

	for operatingSystem, gameAssets := range osGameAssets {

		for _, gameAsset := range gameAssets {

			catalogItem, err := egsGetCatalogItem(&gameAsset, ii, rdx)
			if err != nil {
				return nil, err
			}

			if len(catalogItem.MainGameItemList) > 0 {
				continue
			}

			if index := availableProductIndex(gameAsset.AppName, availableProducts); index != -1 {
				availableProducts[index].OperatingSystems = append(availableProducts[index].OperatingSystems, operatingSystem)
			} else {
				ap := vangogh_integration.AvailableProduct{
					Id:               gameAsset.AppName,
					Title:            catalogItem.Title,
					OperatingSystems: []vangogh_integration.OperatingSystem{operatingSystem},
				}

				var dlcGameAssets map[string]string
				dlcGameAssets, err = egsCatalogItemDlcGameAssets(osGameAssets, operatingSystem, catalogItem, ii.force)
				if err != nil {
					return nil, err
				}

				ap.Dlc = dlcGameAssets

				availableProducts = append(availableProducts, ap)
			}
		}
	}

	if ii.OperatingSystem != vangogh_integration.AnyOperatingSystem {
		osAvailableProducts := make([]vangogh_integration.AvailableProduct, 0, len(availableProducts))
		for _, ap := range availableProducts {
			if slices.Contains(ap.OperatingSystems, ii.OperatingSystem) {
				osAvailableProducts = append(osAvailableProducts, ap)
			}
		}
		availableProducts = osAvailableProducts
	}

	return availableProducts, nil
}

func egsReadLocalGameAssets(operatingSystem vangogh_integration.OperatingSystem) ([]egs_integration.GameAsset, error) {

	if !slices.Contains(egs_integration.SupportedOperatingSystems, operatingSystem) {
		return nil, operatingSystem.ErrUnsupported()
	}

	egsOsApKey := originAvailableProductsKey(data.EpicGamesOrigin, operatingSystem)

	availableProductsDir := data.Pwd.AbsRelDirPath(data.AvailableProducts, data.Metadata)
	kvAvailableProducts, err := kevlar.New(availableProductsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if !kvAvailableProducts.Has(egsOsApKey) {
		return nil, nil
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

func egsFetchGameAssets(operatingSystem vangogh_integration.OperatingSystem) error {

	efapa := nod.Begin(" fetching EGS game assets...")
	defer efapa.Done()

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	ptr, err := egsVerifyToken(client)
	if err != nil {
		return err
	}

	rcGameAssets, err := egs_integration.GetGameAssets(egs_integration.Platform(operatingSystem), ptr.AccessToken, client)
	if err != nil {
		return err
	}

	defer rcGameAssets.Close()

	egsOsApKey := originAvailableProductsKey(data.EpicGamesOrigin, operatingSystem)

	availableProductsDir := data.Pwd.AbsRelDirPath(data.AvailableProducts, data.Metadata)
	kvAvailableProducts, err := kevlar.New(availableProductsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	return kvAvailableProducts.Set(egsOsApKey, rcGameAssets)
}

func egsGetCatalogItem(gameAsset *egs_integration.GameAsset, ii *InstallInfo, rdx redux.Writeable) (*egs_integration.CatalogItem, error) {

	catalogItemsDir := data.Pwd.AbsRelDirPath(data.CatalogItems, data.Metadata)
	kvCatalogItems, err := kevlar.New(catalogItemsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if !kvCatalogItems.Has(gameAsset.CatalogItemId) || ii.force {

		if err = egsFetchCatalogItem(gameAsset, kvCatalogItems, rdx); err != nil {
			return nil, err
		}
	}

	return egsReadLocalCatalogItem(gameAsset.CatalogItemId, kvCatalogItems)
}

func egsFetchCatalogItem(gameAsset *egs_integration.GameAsset, kvCatalogItems kevlar.KeyValues, rdx redux.Writeable) error {

	efcia := nod.Begin(" fetching catalog item %s...", gameAsset.CatalogItemId)
	defer efcia.Done()

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	ptr, err := egsVerifyToken(client)
	if err != nil {
		return err
	}

	rcCatalogItem, err := egs_integration.GetCatalogItem(gameAsset.Namespace, gameAsset.CatalogItemId, ptr.AccessToken, client)
	if err != nil {
		return err
	}

	defer rcCatalogItem.Close()

	if err = kvCatalogItems.Set(gameAsset.CatalogItemId, rcCatalogItem); err != nil {
		return err
	}

	return egsReduceCatalogItem(gameAsset.AppName, gameAsset.CatalogItemId, kvCatalogItems, rdx)
}

func egsReadLocalCatalogItem(catalogItemId string, kvCatalogItems kevlar.KeyValues) (*egs_integration.CatalogItem, error) {

	rcCatalogItem, err := kvCatalogItems.Get(catalogItemId)
	if err != nil {
		return nil, err
	}
	defer rcCatalogItem.Close()

	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, rcCatalogItem); err != nil {
		return nil, err
	}

	jsonCatalogItem := strings.HasPrefix(buf.String(), jsonCatalogItemPfx)

	switch jsonCatalogItem {
	case true:
		var catalogItem egs_integration.CatalogItem
		if err = json.UnmarshalRead(buf, &catalogItem); err != nil {
			return nil, err
		}
		return &catalogItem, nil
	default:
		var catalogItemMap map[string]egs_integration.CatalogItem
		if err = json.UnmarshalRead(buf, &catalogItemMap); err != nil {
			return nil, err
		}
		return new(catalogItemMap[catalogItemId]), nil
	}
}

func egsGetGameAsset(appName string, ii *InstallInfo) (*egs_integration.GameAsset, error) {

	egga := nod.Begin("getting EGS game asset...")
	defer egga.Done()

	switch appName {
	case eosOverlayGameAsset.AppName:
		return &eosOverlayGameAsset, nil
	case eosHelperGameAsset.AppName:
		return &eosHelperGameAsset, nil
	default:
		// proceed normally
	}

	osGameAssets, err := egsGetGameAssets(ii.force)
	if err != nil {
		return nil, err
	}

	for sos, gameAssets := range osGameAssets {
		if ii.OperatingSystem != vangogh_integration.AnyOperatingSystem && ii.OperatingSystem != sos {
			continue
		}
		for _, gameAsset := range gameAssets {
			if gameAsset.AppName == appName {
				return &gameAsset, nil
			}
		}
	}

	return nil, errors.New("game asset not found for appName " + appName)
}

func egsGetGameManifest(gameAsset *egs_integration.GameAsset, ii *InstallInfo, force bool) (*egs_integration.GameManifest, error) {

	eggma := nod.Begin("getting EGS game manifest...")
	defer eggma.Done()

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

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	ptr, err := egsVerifyToken(client)
	if err != nil {
		return err
	}

	rcGameManifest, err := egs_integration.GetGameManifest(
		gameAsset.Namespace,
		gameAsset.CatalogItemId,
		gameAsset.AppName,
		egs_integration.Platform(operatingSystem),
		ptr.AccessToken, client)
	if err != nil {
		return err
	}

	defer rcGameManifest.Close()

	return kvGameManifests.Set(key, rcGameManifest)
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

	return egs_integration.ReadManifest(manifestFile)
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

func egsReduceCatalogItem(appName, catalogItemId string, kvCatalogItems kevlar.KeyValues, rdx redux.Writeable) error {

	if err := rdx.MustHave(vangogh_integration.TitleProperty, vangogh_integration.RequiresGamesProperty); err != nil {
		return err
	}

	rcCatalogItem, err := kvCatalogItems.Get(catalogItemId)
	if err != nil {
		return err
	}

	defer rcCatalogItem.Close()

	var catalogItemMap map[string]egs_integration.CatalogItem
	if err = json.UnmarshalRead(rcCatalogItem, &catalogItemMap); err != nil {
		return err
	}

	catalogItem := catalogItemMap[catalogItemId]

	if err = rdx.ReplaceValues(vangogh_integration.TitleProperty, appName, catalogItem.Title); err != nil {
		return err
	}

	if len(catalogItem.MainGameItemList) > 0 {
		var mainGameItems []string

		for _, mainGameItem := range catalogItem.MainGameItemList {
			for _, releaseInfo := range mainGameItem.ReleaseInfo {
				mainGameItems = append(mainGameItems, releaseInfo.AppId)
			}
		}

		if len(mainGameItems) > 0 {
			if err = rdx.ReplaceValues(vangogh_integration.RequiresGamesProperty, appName, mainGameItems...); err != nil {
				return err
			}
		}
	}

	return nil
}

func egsRemoveChunks(appName string, operatingSystem vangogh_integration.OperatingSystem, originData *data.OriginData) error {

	erca := nod.NewProgress(" removing EGS chunks...")
	defer erca.Done()

	erca.TotalInt(len(originData.Manifest.ChunkList.Chunks))

	featureLevel := originData.Manifest.Metadata.FeatureLevel
	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, operatingSystem)

	for _, chunk := range originData.Manifest.ChunkList.Chunks {
		absChunkPath := filepath.Join(absChunksDownloadsDir, chunk.Path(featureLevel))
		if _, err := os.Stat(absChunkPath); os.IsNotExist(err) {
			erca.Increment()
			continue
		}
		if err := os.Remove(absChunkPath); err != nil {
			return err
		}
		erca.Increment()
	}

	return os.RemoveAll(absChunksDownloadsDir)
}

func egsUninstall(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	eua := nod.NewProgress("uninstalling EGS %s...", appName)
	defer eua.Done()

	eua.TotalInt(len(originData.Manifest.FileList.List))

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return err
	}

	for _, file := range originData.Manifest.FileList.List {
		absFilePath := filepath.Join(installedPath, file.Filename)
		if _, err = os.Stat(absFilePath); os.IsNotExist(err) {
			eua.Increment()
			continue
		}
		if err = os.Remove(absFilePath); err != nil {
			return err
		}
	}

	var installedFiles []string
	if installedFiles, err = relWalkDir(installedPath); err == nil && len(installedFiles) == 0 {
		if err = os.RemoveAll(installedPath); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func egsCatalogItemAssets(catalogItem *egs_integration.CatalogItem) (map[steam_grid.Asset]*url.URL, error) {

	shortcutAssets := make(map[steam_grid.Asset]*url.URL)

	for _, keyImage := range catalogItem.KeyImages {

		var asset steam_grid.Asset

		switch keyImage.Type {
		case "DieselGameBox":
			asset = steam_grid.Header
		case "DieselGameBoxTall":
			asset = steam_grid.LibraryCapsule
		}

		if u, err := url.Parse(keyImage.Url); err == nil {
			shortcutAssets[asset] = u
		} else {
			return nil, err
		}
	}

	return shortcutAssets, nil
}

func egsAssembleChunks(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	eaca := nod.NewProgress("assembling EGS chunks into files for %s-%s...", appName, ii.OperatingSystem)
	defer eaca.Done()

	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, ii.OperatingSystem)

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return err
	}

	eaca.Total(uint64(egsManifestSize(originData.Manifest)))

	for _, chunkedFile := range originData.Manifest.FileList.List {
		if err = egsAssembleFile(&chunkedFile, originData.Manifest.Metadata.FeatureLevel, absChunksDownloadsDir, installedPath); err != nil {
			return err
		}

		eaca.Progress(chunkedFile.Size)
	}

	return nil
}

func egsAssembleFile(chunkedFile *egs_integration.File, featureLevel uint32, chunksDir, installedPath string) error {

	var err error

	absOutputFilename := filepath.Join(installedPath, chunkedFile.Filename)
	absOutputDir, _ := filepath.Split(absOutputFilename)

	if _, err = os.Stat(absOutputDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absOutputDir, 0775); err != nil {
			return err
		}
	}

	outFile, err := os.Create(absOutputFilename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for _, part := range chunkedFile.Parts {

		if err = egsWriteChunkPart(&part, featureLevel, chunksDir, outFile); err != nil {
			return err
		}
	}

	return nil
}

func egsWriteChunkPart(part *egs_integration.ChunkPart, featureLevel uint32, chunksDir string, outFile *os.File) error {

	chunkPath := filepath.Join(chunksDir, part.Chunk.Path(featureLevel))

	var chunkFile *os.File
	chunkFile, err := os.Open(chunkPath)
	if err != nil {
		return err
	}
	defer chunkFile.Close()

	var chunkReader io.Reader
	chunkReader, err = egs_integration.ReadChunk(chunkFile)
	if err != nil {
		return nil
	}

	var chunkData []byte
	chunkData, err = io.ReadAll(chunkReader)
	if err != nil {
		return err
	}

	if _, err = io.Copy(outFile, bytes.NewReader(chunkData[part.Offset:part.Offset+part.Size])); err != nil {
		return err
	}

	return nil
}

func egsValidateAssembly(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	evaa := nod.NewProgress("validating assembled files for %s-%s...", appName, ii.Origin)
	defer evaa.Done()

	evaa.Total(uint64(egsManifestSize(originData.Manifest)))

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return err
	}

	for _, file := range originData.Manifest.FileList.List {
		if err = egsValidateAssembledFile(installedPath, &file); err != nil {
			return err
		}

		evaa.Progress(file.Size)
	}

	return nil
}

func egsValidateAssembledFile(installedDir string, assembledFile *egs_integration.File) error {

	var err error

	absFilename := filepath.Join(installedDir, assembledFile.Filename)

	inputFile, err := os.Open(absFilename)
	if err != nil {
		return err
	}

	shaSum := sha1.New()

	if _, err = io.Copy(shaSum, inputFile); err != nil {
		return err
	}

	actualShaSum := fmt.Sprintf("%x", shaSum.Sum(nil))
	expectedShaSum := fmt.Sprintf("%x", assembledFile.ShaHash)

	if actualShaSum != expectedShaSum {
		return errors.New("failed validation for " + assembledFile.Filename)
	}

	return nil
}

func egsChmodLauncherExe(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	switch ii.OperatingSystem {

	case vangogh_integration.MacOS:

		installedPath, err := originOsInstalledPath(id, ii, rdx)
		if err != nil {
			return err
		}

		manifestLaunchExe := originData.Manifest.Metadata.LaunchExe

		absLaunchExePath := filepath.Join(installedPath, manifestLaunchExe)

		if _, err = os.Stat(absLaunchExePath); err == nil {
			if err = chmodExecutable(absLaunchExePath); err != nil {
				return err
			}
		}
	default:
		// do nothing
	}

	return nil
}

func egsManifestVersion(manifest *egs_integration.Manifest) string {
	if manifest != nil &&
		manifest.Metadata != nil {
		return manifest.Metadata.BuildVersion
	}
	return ""
}

func egsManifestSize(manifest *egs_integration.Manifest) int64 {
	var totalEstimatedBytes int64

	for _, file := range manifest.FileList.List {
		totalEstimatedBytes += int64(file.Size)
	}

	return totalEstimatedBytes
}

func egsAssembleValidateChunks(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	egsAppsDir := data.Pwd.AbsDirPath(data.EgsApps)

	if err := originHasFreeSpace(appName, egsAppsDir, ii, originData); err != nil {
		return err
	}

	if err := egsAssembleChunks(appName, ii, originData, rdx); err != nil {
		return err
	}

	if err := egsValidateAssembly(appName, ii, originData, rdx); err != nil {
		return err
	}

	return nil
}

func egsContainsGameAsset(appName string, gameAssets []egs_integration.GameAsset) bool {
	for _, ga := range gameAssets {
		if ga.AppName == appName {
			return true
		}
	}
	return false
}

func egsCatalogItemDlcGameAssets(osGameAssets map[vangogh_integration.OperatingSystem][]egs_integration.GameAsset, operatingSystem vangogh_integration.OperatingSystem, catalogItem *egs_integration.CatalogItem, force bool) (map[string]string, error) {

	dlcGameAssets := make(map[string]string)

	if len(catalogItem.DlcItemList) == 0 {
		return dlcGameAssets, nil
	}

	for gaOs, gameAssets := range osGameAssets {
		if gaOs != operatingSystem {
			continue
		}

		for _, dlcItem := range catalogItem.DlcItemList {
			for _, releaseInfo := range dlcItem.ReleaseInfo {
				if egsContainsGameAsset(releaseInfo.AppId, gameAssets) {
					dlcGameAssets[releaseInfo.AppId] = dlcItem.Title
				}
			}
		}

	}

	return dlcGameAssets, nil
}

func egsInstallDownloadableContent(ii *InstallInfo, catalogItem *egs_integration.CatalogItem) error {

	if !slices.Contains(ii.DownloadTypes, vangogh_integration.DLC) {
		return nil
	}

	if len(catalogItem.DlcItemList) == 0 {
		return nil
	}

	eidca := nod.Begin("installing available DLCs for %s...", catalogItem.Title)
	defer eidca.Done()

	osGameAssets, err := egsGetGameAssets(ii.force)
	if err != nil {
		return err
	}

	dlcGameAssets, err := egsCatalogItemDlcGameAssets(osGameAssets, ii.OperatingSystem, catalogItem, ii.force)
	if err != nil {
		return err
	}

	for dlcAppName, dlcTitle := range dlcGameAssets {
		if err = Install(dlcAppName, ii); err != nil {
			return err
		}

		ii.DownloadableContent = append(ii.DownloadableContent, dlcTitle)
	}

	return nil
}

func egsUninstallDownloadableContent(appName string, ii *InstallInfo, rdx redux.Writeable) error {

	eudca := nod.Begin("uninstalling DLCs for %s...", appName)
	defer eudca.Done()

	gameAsset, err := egsGetGameAsset(appName, ii)
	if err != nil {
		return err
	}

	catalogItem, err := egsGetCatalogItem(gameAsset, ii, rdx)
	if err != nil {
		return err
	}

	osGameAssets, err := egsGetGameAssets(ii.force)
	if err != nil {
		return err
	}

	catalogItemDlcs, err := egsCatalogItemDlcGameAssets(osGameAssets, ii.OperatingSystem, catalogItem, ii.force)
	for dlcItemId := range catalogItemDlcs {
		if err = originUninstall(dlcItemId, ii, rdx); err != nil {
			return err
		}
	}

	return nil
}

func egsValidateChunks(appName string, ii *InstallInfo, originData *data.OriginData) error {

	evca := nod.NewProgress("validating EGS chunks for %s-%s...", appName, ii.OperatingSystem)
	defer evca.Done()

	evca.Total(uint64(egsManifestSize(originData.Manifest)))

	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, ii.OperatingSystem)

	for _, chunk := range originData.Manifest.ChunkList.Chunks {

		chunkPath := chunk.Path(originData.Manifest.Metadata.FeatureLevel)

		absChunkFilename := filepath.Join(absChunksDownloadsDir, chunkPath)

		chunkFile, err := os.Open(absChunkFilename)
		if err != nil {
			return err
		}

		chunkReader, err := egs_integration.ReadChunk(chunkFile)
		if err != nil {
			return err
		}

		shaSum := sha1.New()

		if _, err = io.Copy(shaSum, chunkReader); err != nil {
			return err
		}

		expectedShaSum := fmt.Sprintf("%x", chunk.ShaHash)
		actualShaSum := fmt.Sprintf("%x", shaSum.Sum(nil))

		if expectedShaSum != actualShaSum {
			return errors.New("failed validation for " + chunkPath)
		}

		evca.Progress(chunk.FileSize)
	}

	evca.EndWithResult("valid")

	return nil
}

func egsSetupConnection(cookieStr string, reset bool) error {

	egsca := nod.Begin("connecting to EGS...")
	defer egsca.Done()

	var err error

	if reset {
		if err = egsResetConnection(); err != nil {
			return err
		}
	}

	if cookieStr != "" {
		if err = egsGetAccessToken(cookieStr); err != nil {
			return err
		}
	}

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	_, err = egsVerifyToken(client)
	return err
}

func egsResetConnection() error {

	egrc := nod.Begin("resetting EGS connection...")
	defer egrc.Done()

	cookiesDir := data.Pwd.AbsRelDirPath(data.Cookies, data.Metadata)
	egsCookiePath := filepath.Join(cookiesDir, egsCookiesFilename)

	tokensDir := data.Pwd.AbsRelDirPath(data.Tokens, data.Metadata)
	kvTokens, err := kevlar.New(tokensDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if _, err = os.Stat(egsCookiePath); err == nil {
		if err = os.Remove(egsCookiePath); err != nil {
			return nil
		}
	}

	if kvTokens.Has(egsTokenKey) {
		if err = kvTokens.Cut(egsTokenKey); err != nil {
			return err
		}
	}

	return nil
}

func egsDownloadChunks(appName string, ii *InstallInfo, originData *data.OriginData) error {

	edca := nod.NewProgress("downloading EGS chunks...")
	edca.Done()

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	if err := originHasFreeSpace(appName, downloadsDir, ii, originData); err != nil {
		return err
	}

	edca.Total(uint64(egsManifestSize(originData.Manifest)))

	cdnUrls, err := originData.GameManifest.Urls()
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	var cdnUrl *url.URL
	for _, cu := range cdnUrls {
		cdnUrl = cu
		break
	}

	if cdnUrl == nil {
		return errors.New("downloading EGS chunks requires CDN url")
	}

	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, ii.OperatingSystem)

	originalPath := strings.TrimSuffix(cdnUrl.Path, filepath.Base(cdnUrl.Path))
	cdnUrl.RawQuery = ""

	for _, chunk := range originData.Manifest.ChunkList.Chunks {

		chunkPath := chunk.Path(originData.Manifest.Metadata.FeatureLevel)
		cdnUrl.Path = path.Join(originalPath, chunkPath)

		if err = dc.Download(cdnUrl, ii.force, nil, absChunksDownloadsDir, chunkPath); err != nil {
			return err
		}

		edca.Progress(chunk.FileSize)
	}

	return nil
}

func egsGetExecTask(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable, et *execTask) (*execTask, error) {

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return nil, err
	}

	absPrefixDir, err := data.AbsPrefixDir(appName, ii.Origin, rdx)
	if err != nil {
		return nil, err
	}

	launchDir, launchFile := filepath.Split(originData.Manifest.Metadata.LaunchExe)

	et.title = launchFile
	et.prefix = absPrefixDir
	et.exe = filepath.Join(installedPath, originData.Manifest.Metadata.LaunchExe)
	if originData.Manifest.Metadata.LaunchCommand != "" {
		et.args = append(et.args, originData.Manifest.Metadata.LaunchCommand)
	}
	et.workDir = filepath.Join(installedPath, launchDir)

	return et, nil
}
