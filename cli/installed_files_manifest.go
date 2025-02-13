package cli

import (
	"bufio"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/redux"
	"io"
	"os"
	"path/filepath"
	"slices"
)

func createManifest(absRootDir string, id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable, utcTime int64) error {
	relFiles, err := data.GetRelFilesModifiedAfter(absRootDir, utcTime)
	if err != nil {
		return err
	}

	return appendManifest(id, langCode, operatingSystem, rdx, relFiles...)
}

func readManifest(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) ([]string, error) {
	absManifestFilename, err := data.GetAbsManifestFilename(id, langCode, operatingSystem, rdx)
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(absManifestFilename); os.IsNotExist(err) {
		return nil, nil
	}

	manifestFile, err := os.Open(absManifestFilename)
	if err != nil {
		return nil, err
	}

	relFiles := make([]string, 0)
	manifestScanner := bufio.NewScanner(manifestFile)
	for manifestScanner.Scan() {
		relFiles = append(relFiles, manifestScanner.Text())
	}

	if err = manifestScanner.Err(); err != nil {
		return nil, err
	}

	return relFiles, nil
}

func appendManifest(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable, newRelFiles ...string) error {

	absManifestFilename, err := data.GetAbsManifestFilename(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	relFiles, err := readManifest(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	for _, nrf := range newRelFiles {
		if slices.Contains(relFiles, nrf) {
			continue
		}
		relFiles = append(relFiles, nrf)
	}

	absManifestDir, _ := filepath.Split(absManifestFilename)
	if _, err = os.Stat(absManifestDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absManifestDir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	manifestFile, err := os.Create(absManifestFilename)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	for _, relFile := range relFiles {
		if _, err = io.WriteString(manifestFile, relFile+"\n"); err != nil {
			return err
		}
	}

	return nil
}
