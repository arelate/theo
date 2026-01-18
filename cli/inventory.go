package cli

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/redux"
)

func readInventory(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) ([]string, error) {

	absInventoryFilename, err := data.AbsInventoryFilename(id, langCode, operatingSystem, rdx)
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(absInventoryFilename); os.IsNotExist(err) {
		return nil, nil
	}

	inventoryFile, err := os.Open(absInventoryFilename)
	if err != nil {
		return nil, err
	}

	var relFiles []string
	if err = json.NewDecoder(inventoryFile).Decode(&relFiles); err != nil {
		return nil, err
	}

	return relFiles, nil
}

func writeInventory(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable, inventory ...string) error {

	absInventoryFilename, err := data.AbsInventoryFilename(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	absInventoryDir, _ := filepath.Split(absInventoryFilename)

	pathDir, _ := filepath.Split(absInventoryDir)
	if _, err = os.Stat(pathDir); os.IsNotExist(err) {
		if err = os.MkdirAll(pathDir, 0755); err != nil {
			return err
		}
	}

	inventoryFile, err := os.Create(absInventoryFilename)
	if err != nil {
		return err
	}
	defer inventoryFile.Close()

	return json.NewEncoder(inventoryFile).Encode(inventory)
}
