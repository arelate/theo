package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/egs_integration"
	"github.com/boggydigital/coost"
	"github.com/boggydigital/dolo"
)

func TestEpicGamesHandler(u *url.URL) error {

	q := u.Query()

	lsManifests := q.Has("list-manifests")
	dlChunks := q.Has("download-chunks")
	tsChunks := q.Has("test-chunks")

	id := q.Get("id")
	cdnUrlStr := q.Get("cdn-url")

	cdnUrl, err := url.Parse(cdnUrlStr)
	if err != nil {
		return err
	}

	if lsManifests {
		return listManifests(id)
	} else if dlChunks {
		return downloadChunks(id, cdnUrl)
	} else if tsChunks {
		return testChunks(id)
	}

	return errors.New("need apis or manifest")
}

func downloadChunks(manifestId string, cdnUrl *url.URL) error {

	if manifestId == "" {
		return errors.New("empty manifest id")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	manifestFilename := manifestId
	if !strings.HasSuffix(manifestId, ".manifest") {
		manifestFilename += ".manifest"
	}

	absManifestPath := filepath.Join(homeDir, "Downloads", "epic", manifestFilename)

	manifestFile, err := os.Open(absManifestPath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	manifest, err := egs_integration.ReadBinaryManifest(manifestFile)
	if err != nil {
		return err
	}

	originalPath := cdnUrl.Path

	targetChunksDir := filepath.Join(homeDir, "Downloads", "epic", "chunks", strings.TrimSuffix(manifestId, ".manifest"))

	dc := dolo.DefaultClient

	for ii, chk := range manifest.ChunkList.Chunks {
		cdnUrl.Path = path.Join(originalPath, chk.Path(manifest.Metadata.FeatureLevel))

		if err = dc.Download(cdnUrl, false, nil, targetChunksDir); err != nil {
			return err
		}

		fmt.Println(ii, len(manifest.ChunkList.Chunks))
	}

	return nil
}

//func testApis() error {
//
//	//fmt.Println("GetGameAssets")
//	//
//	//gameAssets, err := egs_integration.GetGameAssets("Windows", verifyTokenResponse.Token, client)
//	//if err != nil {
//	//	return err
//	//}
//	//
//	//fmt.Println(gameAssets)
//
//	//fmt.Println("GetLauncherManifests")
//	//
//	//launcherManifests, err := egs_integration.GetLauncherManifests("Windows", verifyTokenResponse.Token, client)
//	//if err != nil {
//	//	return err
//	//}
//	//
//	//fmt.Println(launcherManifests)
//
//	//fmt.Println("GetUserEntitlements")
//	//
//	//entitlements, err := egs_integration.GetUserEntitlements(verifyTokenResponse.AccountId, verifyTokenResponse.Token, 0, 1000, client)
//	//if err != nil {
//	//	return err
//	//}
//	//
//	//for _, ent := range entitlements {
//	//
//	//	var catalogItem *egs_integration.CatalogItem
//	//	catalogItem, err = egs_integration.GetCatalogItem(ent.Namespace, ent.CatalogItemId, verifyTokenResponse.Token, client)
//	//	if err != nil {
//	//		return err
//	//	}
//	//
//	//	fmt.Println(catalogItem)
//	//
//	//}
//
//	fmt.Println("GetLibraryItems")
//
//	libraryItems, err := egs_integration.GetLibraryItems("", verifyTokenResponse.Token, client)
//	if err != nil {
//		return err
//	}
//
//	limit := 10
//
//	for ii, rec := range libraryItems.Records {
//
//		var catalogItem *egs_integration.CatalogItem
//		catalogItem, err = egs_integration.GetCatalogItem(rec.Namespace, rec.CatalogItemId, verifyTokenResponse.Token, client)
//		if err != nil {
//			return err
//		}
//
//		//fmt.Println(catalogItem)
//		fmt.Println(catalogItem.Title)
//
//		var gameManifest *egs_integration.GameManifest
//		gameManifest, err = egs_integration.GetGameManifest(rec.Namespace, rec.CatalogItemId, rec.AppName, "Windows", verifyTokenResponse.Token, client)
//		if err != nil {
//			return err
//		}
//
//		//fmt.Println(gameManifest.Elements)
//
//		for _, element := range gameManifest.Elements {
//			for _, manifest := range element.Manifests {
//				var manifestUrl *url.URL
//				manifestUrl, err = url.Parse(manifest.Uri)
//				if err != nil {
//					return err
//				}
//
//				q := manifestUrl.Query()
//
//				for _, kv := range manifest.QueryParams {
//					q.Add(kv.Name, kv.Value)
//				}
//
//				manifestUrl.RawQuery = q.Encode()
//
//				fmt.Println(" - " + manifestUrl.String())
//			}
//		}
//
//		if ii == limit-1 {
//			break
//		}
//
//	}
//
//	//fmt.Println("GetGameAssets")
//	//
//	//var gameAssets []egs_integration.GameAsset
//	//gameAssets, err = egs_integration.GetGameAssets("Windows", verifyTokenResponse.Token, client)
//	//if err != nil {
//	//	panic(err)
//	//}
//	//
//	//fmt.Println(gameAssets)
//
//	fmt.Println("DeleteToken")
//
//	if err = egs_integration.DeleteToken(verifyTokenResponse.Token, client); err != nil {
//		return err
//	}
//
//	//fmt.Println("GetEntitlements")
//	//
//	//entitlements, err := egs_integration.GetEntitlements(verifyTokenResponse.AccountId, verifyTokenResponse.Token, client)
//	//if err != nil {
//	//	return err
//	//}
//
//	//
//	//for _, ent := range entitlements {
//	//
//	//	fmt.Println("GetCatalogItem", ent)
//	//
//	//	entStr, err := egs_integration.GetCatalogItem(ent.Namespace, ent.CatalogItemId, postTokenResponse.AccessToken, client)
//	//	if err != nil {
//	//		return err
//	//	}
//	//
//	//	fmt.Println(entStr)
//	//
//	//	break
//	//}
//
//	return nil
//}

func listManifests(appId string) error {

	fmt.Println()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Login to epicgames.com and use a simple URL like https://www.epicgames.com/id/api/redirect
	// to capture cookies using instructions from https://github.com/boggydigital/coost
	importCookiesPath := filepath.Join(homeDir, "Downloads", "epic", "import_cookies.txt")

	if _, err = os.Stat(importCookiesPath); err != nil {
		return err
	}

	outputCookiesPath := filepath.Join(homeDir, "Downloads", "epic", "egs_integration_cookies.json")

	var importCookieBytes []byte
	importCookieBytes, err = os.ReadFile(importCookiesPath)
	if err != nil {
		return err
	}

	if err = coost.Import(string(importCookieBytes), egs_integration.HostUrl(), outputCookiesPath); err != nil {
		return err
	}

	jar, err := coost.Read(egs_integration.HostUrl(), outputCookiesPath)
	if err != nil {
		return err
	}

	client := http.DefaultClient
	client.Jar = jar

	fmt.Println("GetApiRedirect")

	apiRedirectResponse, err := egs_integration.GetApiRedirect(client)
	if err != nil {
		return err
	}

	fmt.Println("PostToken")

	postTokenResponse, err := egs_integration.PostToken(apiRedirectResponse.AuthorizationCode, client)
	if err != nil {
		return err
	}

	if postTokenResponse.AccessToken == "" {
		return errors.New("failed to get access token")
	}

	fmt.Println("GetVerifyToken")

	verifyTokenResponse, err := egs_integration.GetVerifyToken(postTokenResponse.AccessToken, client)
	if err != nil {
		return err
	}

	gameAssets, err := egs_integration.GetGameAssets("Windows", verifyTokenResponse.Token, client)
	if err != nil {
		return err
	}

	for _, ga := range gameAssets {

		if ga.AppName != appId {
			continue
		}

		fmt.Println("appname:" + ga.AppName)
		fmt.Println("namespace:" + ga.Namespace)
		fmt.Println("catalog-item:" + ga.CatalogItemId)

		var gameManifest *egs_integration.GameManifest
		gameManifest, err = egs_integration.GetGameManifest(ga.Namespace, ga.CatalogItemId, ga.AppName, "Windows", verifyTokenResponse.Token, client)
		if err != nil {
			return err
		}

		for _, element := range gameManifest.Elements {
			for _, manifest := range element.Manifests {
				var mu *url.URL
				mu, err = manifest.Url()
				if err != nil {
					return err
				}
				fmt.Println(" - " + mu.String())
			}
		}

	}

	return nil
}

func testChunks(manifestId string) error {

	if manifestId == "" {
		return errors.New("empty manifest id")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	manifestFilename := manifestId
	if !strings.HasSuffix(manifestId, ".manifest") {
		manifestFilename += ".manifest"
	}

	absManifestPath := filepath.Join(homeDir, "Downloads", "epic", manifestFilename)

	manifestFile, err := os.Open(absManifestPath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	manifest, err := egs_integration.ReadBinaryManifest(manifestFile)
	if err != nil {
		return err
	}

	targetChunksDir := filepath.Join(homeDir, "Downloads", "epic", "chunks", strings.TrimSuffix(manifestId, ".manifest"))

	for _, chk := range manifest.ChunkList.Chunks {

		chkFilename := filepath.Base(chk.Path(manifest.Metadata.FeatureLevel))

		var chunkFile *os.File
		chunkFile, err = os.Open(filepath.Join(targetChunksDir, chkFilename))
		if err != nil {
			return err
		}

		//var chkReader io.ReadSeeker
		_, err = egs_integration.ReadChunk(chunkFile)
		if err != nil {
			return err
		}

		if err = chunkFile.Close(); err != nil {
			return err
		}

		fmt.Println(filepath.Join(targetChunksDir, chkFilename))

		break
	}

	return nil
}
