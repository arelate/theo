package cli

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
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
	vlChunks := q.Has("validate-chunks")
	vlFiles := q.Has("validate-files")
	asChunks := q.Has("assemble-chunks")

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
	} else if vlChunks {
		return validateChunks(id)
	} else if asChunks {
		return assembleChunks(id)
	} else if vlFiles {
		return validateFiles(id)
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

		if err = dc.Download(cdnUrl, false, nil, targetChunksDir, chk.Path(manifest.Metadata.FeatureLevel)); err != nil {
			return err
		}

		fmt.Println(ii, len(manifest.ChunkList.Chunks))
	}

	return nil
}

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

	postTokenResponse, err := egs_integration.PostToken(apiRedirectResponse.AuthorizationCode, egs_integration.GrantTypeAuthorizationCode, client)
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

	gameAssets, err := egs_integration.GetGameAssets("Mac", verifyTokenResponse.Token, client)
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
		gameManifest, err = egs_integration.GetGameManifest(ga.Namespace, ga.CatalogItemId, ga.AppName, "Mac", verifyTokenResponse.Token, client)
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

func validateChunks(manifestId string) error {
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

	chunksDir := filepath.Join(homeDir, "Downloads", "epic", "chunks", strings.TrimSuffix(manifestId, ".manifest"))

	fmt.Println()

	for _, chunk := range manifest.ChunkList.Chunks {

		chunkPath := filepath.Join(chunksDir, filepath.Base(chunk.Path(manifest.Metadata.FeatureLevel)))

		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return err
		}

		//fmt.Println(chunk.Path(manifest.Metadata.FeatureLevel))
		//fmt.Printf("hash:%x\n", chunk.ShaHash)

		chunkReader, err := egs_integration.ReadChunk(chunkFile)
		if err != nil {
			return err
		}

		chunkData, err := io.ReadAll(chunkReader)
		if err != nil {
			return err
		}

		shaSum := sha1.Sum(chunkData)

		expectedShaSum := fmt.Sprintf("%x", chunk.ShaHash)
		actualShaSum := fmt.Sprintf("%x", shaSum)

		result := "error"
		if expectedShaSum == actualShaSum {
			result = "valid"
		}

		fmt.Println(filepath.Base(chunk.Path(manifest.Metadata.FeatureLevel)), result)
	}

	return nil
}

func assembleChunks(manifestId string) error {

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

	chunksDir := filepath.Join(homeDir, "Downloads", "epic", "chunks", strings.TrimSuffix(manifestId, ".manifest"))

	fmt.Println()

	for _, file := range manifest.FileList.List {
		if err = assembleFile(manifestId, &file, manifest.Metadata.FeatureLevel, chunksDir); err != nil {
			return err
		}
	}

	return nil
}

func assembleFile(manifestId string, f *egs_integration.File, featureLevel uint32, chunksDir string) error {

	//if !strings.HasSuffix(f.Filename, ".json") {
	//	return nil
	//}

	var err error

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	outputDir := filepath.Join(homeDir, "Downloads", "epic", "output", strings.TrimSuffix(manifestId, ".manifest"))

	absOutputFilename := filepath.Join(outputDir, f.Filename)
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

	for _, part := range f.Parts {

		chunkPath := filepath.Join(chunksDir, filepath.Base(part.Chunk.Path(featureLevel)))
		var chunkFile *os.File
		chunkFile, err = os.Open(chunkPath)
		if err != nil {
			return err
		}

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
	}

	return nil
}

func validateFiles(manifestId string) error {

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

	fmt.Println()

	for _, file := range manifest.FileList.List {
		if err = validateFile(manifestId, &file); err != nil {
			return err
		}
	}

	return nil
}

func validateFile(manifestId string, f *egs_integration.File) error {

	var err error

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	outputDir := filepath.Join(homeDir, "Downloads", "epic", "output", strings.TrimSuffix(manifestId, ".manifest"))

	absFilename := filepath.Join(outputDir, f.Filename)

	inputFile, err := os.Open(absFilename)
	if err != nil {
		return err
	}

	inputData, err := io.ReadAll(inputFile)
	if err != nil {
		return err
	}

	shaSum := sha1.Sum(inputData)
	actualShaSum := fmt.Sprintf("%x", shaSum)
	expectedShaSum := fmt.Sprintf("%x", f.ShaHash)

	result := "error"
	if actualShaSum == expectedShaSum {
		result = "valid"
	}

	fmt.Println(f.Filename, result)

	return nil
}
