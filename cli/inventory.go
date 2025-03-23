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

func createInventory(absRootDir string, id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable, utcTime int64) error {
	relFiles, err := data.GetRelFilesModifiedAfter(absRootDir, utcTime)
	if err != nil {
		return err
	}

	return appendInventory(id, langCode, operatingSystem, rdx, relFiles...)
}

func readInventory(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) ([]string, error) {
	absInventoryFilename, err := data.GetAbsInventoryFilename(id, langCode, operatingSystem, rdx)
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(absInventoryFilename); os.IsNotExist(err) {
		return nil, nil
	}

	manifestFile, err := os.Open(absInventoryFilename)
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

func appendInventory(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable, newRelFiles ...string) error {

	absInventoryFilename, err := data.GetAbsInventoryFilename(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	relFiles, err := readInventory(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	for _, nrf := range newRelFiles {
		if slices.Contains(relFiles, nrf) {
			continue
		}
		relFiles = append(relFiles, nrf)
	}

	absInventoryDir, _ := filepath.Split(absInventoryFilename)
	if _, err = os.Stat(absInventoryDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absInventoryDir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	inventoryFile, err := os.Create(absInventoryFilename)
	if err != nil {
		return err
	}
	defer inventoryFile.Close()

	for _, relFile := range relFiles {
		if _, err = io.WriteString(inventoryFile, relFile+"\n"); err != nil {
			return err
		}
	}

	return nil
}
