package cli

import (
	"encoding/json/v2"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func isLinkExecutable(link *vangogh_integration.ProductDownloadLink, operatingSystem vangogh_integration.OperatingSystem) bool {
	ext := filepath.Ext(link.LocalFilename)
	switch operatingSystem {
	case vangogh_integration.MacOS:
		return ext == pkgExt
	case vangogh_integration.Linux:
		return ext == shExt
	case vangogh_integration.Windows:
		return ext == exeExt
	default:
		return false
	}
}

func getInventory(operatingSystem vangogh_integration.OperatingSystem, dls vangogh_integration.ProductDownloadLinks, unpackDir string) ([]string, error) {

	gia := nod.Begin(" creating inventory of unpacked files...")
	defer gia.Done()

	filesMap := make(map[string]any)

	for _, link := range dls {

		if !isLinkExecutable(&link, operatingSystem) {
			continue
		}

		absUnpackedPath := filepath.Join(unpackDir, link.LocalFilename)

		relUnpackedFiles, err := relWalkDir(absUnpackedPath)
		if err != nil {
			return nil, err
		}

		for _, ruf := range relUnpackedFiles {
			filesMap[ruf] = nil
		}

	}

	return slices.Sorted(maps.Keys(filesMap)), nil
}

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
	if err = json.UnmarshalRead(inventoryFile, &relFiles); err != nil {
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

	return json.MarshalWrite(inventoryFile, inventory)
}

func removeInventoriedFiles(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) error {

	umpa := nod.Begin(" removing inventoried files for %s %s-%s...", id, operatingSystem, langCode)
	defer umpa.Done()

	absInstalledPath, err := osInstalledPath(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absInstalledPath); os.IsNotExist(err) {
		umpa.EndWithResult("not present")
		return nil
	}

	relInventory, err := readInventory(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	for _, rif := range relInventory {
		absRif := filepath.Join(absInstalledPath, rif)
		if _, err = os.Stat(absRif); os.IsNotExist(err) {
			continue
		}
		if err = os.Remove(absRif); err != nil {
			return err
		}
	}

	relFiles, err := relWalkDir(absInstalledPath)
	if err != nil {
		return err
	}

	if len(relFiles) == 0 {
		if err = os.RemoveAll(absInstalledPath); err != nil {
			return err
		}
	}

	return nil
}
