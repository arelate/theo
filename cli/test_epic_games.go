package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/arelate/southern_light/epic_games"
	"github.com/boggydigital/coost"
)

func TestEpicGamesHandler(_ *url.URL) error {

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

	outputCookiesPath := filepath.Join(homeDir, "Downloads", "epic_games_cookies.json")

	var importCookieBytes []byte
	importCookieBytes, err = os.ReadFile(importCookiesPath)
	if err != nil {
		return err
	}

	if err = coost.Import(string(importCookieBytes), epic_games.HostUrl(), outputCookiesPath); err != nil {
		return err
	}

	jar, err := coost.Read(epic_games.HostUrl(), outputCookiesPath)
	if err != nil {
		return err
	}

	client := http.DefaultClient
	client.Jar = jar

	fmt.Println("GetApiRedirect")

	apiRedirectResponse, err := epic_games.GetApiRedirect(client)
	if err != nil {
		return err
	}

	fmt.Println("PostToken")

	postTokenResponse, err := epic_games.PostToken(apiRedirectResponse.AuthorizationCode, client)
	if err != nil {
		return err
	}

	fmt.Println("GetVerifyToken")

	verifyTokenResponse, err := epic_games.GetVerifyToken(postTokenResponse.AccessToken, client)
	if err != nil {
		return err
	}

	//fmt.Println("GetGameAssets")
	//
	//gameAssets, err := epic_games.GetGameAssets("Windows", verifyTokenResponse.Token, client)
	//if err != nil {
	//	return err
	//}
	//
	//fmt.Println(gameAssets)

	//fmt.Println("GetLauncherManifests")
	//
	//launcherManifests, err := epic_games.GetLauncherManifests("Windows", verifyTokenResponse.Token, client)
	//if err != nil {
	//	return err
	//}
	//
	//fmt.Println(launcherManifests)

	//fmt.Println("GetUserEntitlements")
	//
	//entitlements, err := epic_games.GetUserEntitlements(verifyTokenResponse.AccountId, verifyTokenResponse.Token, 0, 1000, client)
	//if err != nil {
	//	return err
	//}
	//
	//for _, ent := range entitlements {
	//
	//	var catalogItem *epic_games.CatalogItem
	//	catalogItem, err = epic_games.GetCatalogItem(ent.Namespace, ent.CatalogItemId, verifyTokenResponse.Token, client)
	//	if err != nil {
	//		return err
	//	}
	//
	//	fmt.Println(catalogItem)
	//
	//}

	//fmt.Println("GetLibraryItems")
	//
	//libraryItems, err := epic_games.GetLibraryItems("", verifyTokenResponse.Token, client)
	//if err != nil {
	//	return err
	//}

	//for _, rec := range libraryItems.Records {

	//var catalogItem *epic_games.CatalogItem
	//catalogItem, err = epic_games.GetCatalogItem(rec.Namespace, rec.CatalogItemId, verifyTokenResponse.Token, client)
	//if err != nil {
	//	return err
	//}
	//
	//fmt.Println(catalogItem)

	//var gameManifest *epic_games.GameManifest
	//gameManifest, err = epic_games.GetGameManifest(rec.Namespace, rec.CatalogItemId, rec.AppName, "Windows", verifyTokenResponse.Token, client)
	//if err != nil {
	//	return err
	//}
	//
	//fmt.Println(gameManifest)

	//}

	fmt.Println("GetGameAssets")

	var gameAssets []epic_games.GameAsset
	gameAssets, err = epic_games.GetGameAssets("Windows", verifyTokenResponse.Token, client)
	if err != nil {
		panic(err)
	}

	fmt.Println(gameAssets)

	fmt.Println("DeleteToken")

	if err = epic_games.DeleteToken(verifyTokenResponse.Token, client); err != nil {
		return err
	}

	//fmt.Println("GetEntitlements")
	//
	//entitlements, err := epic_games.GetEntitlements(verifyTokenResponse.AccountId, verifyTokenResponse.Token, client)
	//if err != nil {
	//	return err
	//}

	//
	//for _, ent := range entitlements {
	//
	//	fmt.Println("GetCatalogItem", ent)
	//
	//	entStr, err := epic_games.GetCatalogItem(ent.Namespace, ent.CatalogItemId, postTokenResponse.AccessToken, client)
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
