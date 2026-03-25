package cli

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/egs_integration"
)

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
