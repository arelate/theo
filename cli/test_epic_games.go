package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/arelate/southern_light/egs_integration"
	"github.com/boggydigital/coost"
)

func TestEpicGamesHandler(u *url.URL) error {

	q := u.Query()

	apis := q.Has("apis")
	manifest := q.Has("manifest")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	absManifestPath := filepath.Join(homeDir, "Downloads", "epic.manifest")

	if apis {
		return testApis()
	} else if manifest {
		return testManifest(absManifestPath)
	}

	return errors.New("need apis or manifest")
}

func testManifest(path string) error {

	manifestFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	manifest, err := egs_integration.ReadManifest(manifestFile)
	if err != nil {
		return err
	}

	for _, chk := range manifest.ChunkList.Chunks {
		fmt.Println(manifest.Path(chk))
	}

	fmt.Println(manifest)

	return nil
}

func testApis() error {
	fmt.Println()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Login to epicgames.com and use a simple URL like https://www.epicgames.com/id/api/redirect
	// to capture cookies using instructions from https://github.com/boggydigital/coost
	importCookiesPath := filepath.Join(homeDir, "Downloads", "import_cookies.txt")

	if _, err = os.Stat(importCookiesPath); err != nil {
		return err
	}

	outputCookiesPath := filepath.Join(homeDir, "Downloads", "egs_integration_cookies.json")

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

	//fmt.Println("GetGameAssets")
	//
	//gameAssets, err := egs_integration.GetGameAssets("Windows", verifyTokenResponse.Token, client)
	//if err != nil {
	//	return err
	//}
	//
	//fmt.Println(gameAssets)

	fmt.Println("GetLauncherManifests")

	launcherManifests, err := egs_integration.GetLauncherManifests("Windows", verifyTokenResponse.Token, client)
	if err != nil {
		return err
	}

	fmt.Println(launcherManifests)

	//fmt.Println("GetUserEntitlements")
	//
	//entitlements, err := egs_integration.GetUserEntitlements(verifyTokenResponse.AccountId, verifyTokenResponse.Token, 0, 1000, client)
	//if err != nil {
	//	return err
	//}
	//
	//for _, ent := range entitlements {
	//
	//	var catalogItem *egs_integration.CatalogItem
	//	catalogItem, err = egs_integration.GetCatalogItem(ent.Namespace, ent.CatalogItemId, verifyTokenResponse.Token, client)
	//	if err != nil {
	//		return err
	//	}
	//
	//	fmt.Println(catalogItem)
	//
	//}

	fmt.Println("GetLibraryItems")

	libraryItems, err := egs_integration.GetLibraryItems("", verifyTokenResponse.Token, client)
	if err != nil {
		return err
	}

	for _, rec := range libraryItems.Records {

		var catalogItem *egs_integration.CatalogItem
		catalogItem, err = egs_integration.GetCatalogItem(rec.Namespace, rec.CatalogItemId, verifyTokenResponse.Token, client)
		if err != nil {
			return err
		}

		fmt.Println(catalogItem)

		var gameManifest *egs_integration.GameManifest
		gameManifest, err = egs_integration.GetGameManifest(rec.Namespace, rec.CatalogItemId, rec.AppName, "Windows", verifyTokenResponse.Token, client)
		if err != nil {
			return err
		}

		fmt.Println(gameManifest)

		for _, element := range gameManifest.Elements {
			for _, manifest := range element.Manifests {
				var manifestUrl *url.URL
				manifestUrl, err = url.Parse(manifest.Uri)
				if err != nil {
					return err
				}

				q := manifestUrl.Query()

				for _, kv := range manifest.QueryParams {
					q.Add(kv.Name, kv.Value)
				}

				manifestUrl.RawQuery = q.Encode()

				fmt.Println(manifestUrl.String())
			}
		}

		//break

	}

	//fmt.Println("GetGameAssets")
	//
	//var gameAssets []egs_integration.GameAsset
	//gameAssets, err = egs_integration.GetGameAssets("Windows", verifyTokenResponse.Token, client)
	//if err != nil {
	//	panic(err)
	//}
	//
	//fmt.Println(gameAssets)

	fmt.Println("DeleteToken")

	if err = egs_integration.DeleteToken(verifyTokenResponse.Token, client); err != nil {
		return err
	}

	//fmt.Println("GetEntitlements")
	//
	//entitlements, err := egs_integration.GetEntitlements(verifyTokenResponse.AccountId, verifyTokenResponse.Token, client)
	//if err != nil {
	//	return err
	//}

	//
	//for _, ent := range entitlements {
	//
	//	fmt.Println("GetCatalogItem", ent)
	//
	//	entStr, err := egs_integration.GetCatalogItem(ent.Namespace, ent.CatalogItemId, postTokenResponse.AccessToken, client)
	//	if err != nil {
	//		return err
	//	}
	//
	//	fmt.Println(entStr)
	//
	//	break
	//}

	return nil
}
