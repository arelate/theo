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
	"path/filepath"
	"slices"
	"time"

	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/coost"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

var egsClient *http.Client

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

	var arr *egs_integration.GetApiRedirectResponse
	arr, err = egs_integration.GetApiRedirect(client)
	if err != nil {
		return err
	}

	return egsPostToken(arr.AuthorizationCode, egs_integration.GrantTypeAuthorizationCode)
}

func egsRefreshToken() error {
	erta := nod.Begin("refreshing EGS token...")
	defer erta.Done()

	ptr, err := egsGetStoredPostTokenResponse()
	if err != nil {
		return err
	}

	if ptr.RefreshToken == "" {
		return errors.New("refresh token not present")
	}

	return egsPostToken(ptr.RefreshToken, egs_integration.GrantTypeRefreshToken)
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

	var ptr *egs_integration.PostTokenResponse
	ptr, err = egs_integration.PostToken(token, grantType, client)
	if err != nil {
		return err
	}

	if ptr.AccessToken == "" {
		return errors.New("failed to get EGS access token")
	}

	buf := bytes.NewBuffer(nil)
	if err = json.MarshalWrite(buf, &ptr); err != nil {
		return err
	}

	return kvTokens.Set(egsTokenKey, buf)
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

	var ptr egs_integration.PostTokenResponse
	if err = json.UnmarshalRead(rcEgsToken, &ptr); err != nil {
		return nil, err
	}

	return &ptr, nil
}

func egsVerifyToken() error {

	gvta := nod.Begin("verifying EGS token...")
	defer gvta.Done()

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	var ptr *egs_integration.PostTokenResponse
	if ptr, err = egsGetStoredPostTokenResponse(); err != nil {
		return err
	}

	if ptr.AccessToken == "" {
		return errors.New("empty access token, re-connect EGS")
	}

	if ptr.ExpiresAt.Sub(time.Now()) < time.Minute*30 {
		if err = egsRefreshToken(); err != nil {
			return err
		}
	}

	var vtr *egs_integration.GetVerifyTokenResponse
	vtr, err = egs_integration.GetVerifyToken(ptr.AccessToken, client)
	if err != nil {
		return err
	}

	if vtr.Token == "" {
		return errors.New("empty access token, re-connect EGS")
	}

	return nil
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

			if index := availableProductIndex(gameAsset.AppName, availableProducts); index != -1 {
				availableProducts[index].OperatingSystems = append(availableProducts[index].OperatingSystems, operatingSystem)
			} else {
				ap := vangogh_integration.AvailableProduct{
					Id:               gameAsset.AppName,
					Title:            catalogItem.Title,
					OperatingSystems: []vangogh_integration.OperatingSystem{operatingSystem},
				}

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

	var err error

	var client *http.Client
	if client, err = egsGetClient(); err != nil {
		return err
	}

	if err = egsVerifyToken(); err != nil {
		return err
	}

	ptr, err := egsGetStoredPostTokenResponse()
	if err != nil {
		return err
	}

	gameAssets, err := egs_integration.GetGameAssets(egs_integration.Platform(operatingSystem), ptr.AccessToken, client)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err = json.MarshalWrite(buf, &gameAssets); err != nil {
		return err
	}

	egsOsApKey := originAvailableProductsKey(data.EpicGamesOrigin, operatingSystem)

	availableProductsDir := data.Pwd.AbsRelDirPath(data.AvailableProducts, data.Metadata)
	kvAvailableProducts, err := kevlar.New(availableProductsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	return kvAvailableProducts.Set(egsOsApKey, buf)
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

	if err := egsVerifyToken(); err != nil {
		return err
	}

	ptr, err := egsGetStoredPostTokenResponse()
	if err != nil {
		return err
	}

	client, err := egsGetClient()
	if err != nil {
		return err
	}

	catalogItem, err := egs_integration.GetCatalogItem(gameAsset.Namespace, gameAsset.CatalogItemId, ptr.AccessToken, client)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err = json.MarshalWrite(buf, &catalogItem); err != nil {
		return err
	}

	if err = egsReduceCatalogItem(gameAsset.AppName, catalogItem, rdx); err != nil {
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

	//if err := egsValidateSupportedPlatform(ii); err != nil {
	//	return nil, err
	//}

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

	if err := egsVerifyToken(); err != nil {
		return err
	}

	ptr, err := egsGetStoredPostTokenResponse()
	if err != nil {
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
		ptr.AccessToken, client)
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

func egsReduceCatalogItem(appName string, catalogItem *egs_integration.CatalogItem, rdx redux.Writeable) error {

	if err := rdx.MustHave(vangogh_integration.TitleProperty); err != nil {
		return err
	}

	if err := rdx.ReplaceValues(vangogh_integration.TitleProperty, appName, catalogItem.Title); err != nil {
		return err
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

func egsShortcutAssets(catalogItem *egs_integration.CatalogItem) (map[steam_grid.Asset]*url.URL, error) {

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

	eaca.TotalInt(len(originData.Manifest.FileList.List))

	for _, chunkedFile := range originData.Manifest.FileList.List {
		if err = egsAssembleFile(&chunkedFile, originData.Manifest.Metadata.FeatureLevel, absChunksDownloadsDir, installedPath); err != nil {
			return err
		}

		eaca.Increment()
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

	evaa.TotalInt(len(originData.Manifest.FileList.List))

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return err
	}

	for _, file := range originData.Manifest.FileList.List {
		if err = egsValidateAssembledFile(installedPath, &file); err != nil {
			return err
		}

		evaa.Increment()
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
