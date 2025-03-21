package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"path/filepath"
)

func pinInstalledMetadata(operatingSystems []vangogh_integration.OperatingSystem, langCode string, force bool, ids ...string) error {

	pima := nod.NewProgress("pinning theo metadata as installed...")
	defer pima.Done()

	vangogh_integration.PrintParams(ids, operatingSystems, []string{langCode}, nil, false)

	theoMetadataDir, err := pathways.GetAbsRelDir(data.TheoMetadata)
	if err != nil {
		return err
	}

	kvTheoMetadata, err := kevlar.New(theoMetadataDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	pima.TotalInt(len(ids))

	for _, id := range ids {

		if err = pinTheoMetadata(id, operatingSystems, langCode, kvTheoMetadata, force); err != nil {
			return err
		}

		pima.Increment()

	}

	return nil
}

func pinTheoMetadata(id string,
	operatingSystems []vangogh_integration.OperatingSystem,
	langCode string,
	kvTheoMetadata kevlar.KeyValues,
	force bool) error {

	for _, os := range operatingSystems {
		if err := pinInstalledMetadataForOs(id, os, langCode, kvTheoMetadata, force); err != nil {
			return err
		}
	}

	return nil
}

func pinInstalledMetadataForOs(id string,
	operatingSystem vangogh_integration.OperatingSystem,
	langCode string,
	kvTheoMetadata kevlar.KeyValues,
	force bool) error {

	pimoa := nod.Begin(" pinning metadata as installed...")
	defer pimoa.Done()

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return err
	}

	osLangInstalledMetadataDir := filepath.Join(installedMetadataDir, data.OsLangCode(operatingSystem, langCode))

	kvOsLangInstalledMetadata, err := kevlar.New(osLangInstalledMetadataDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if hasTheoMetadata := kvTheoMetadata.Has(id); !hasTheoMetadata {
		return errors.New("theo metadata not found for: " + id)
	}

	if kvOsLangInstalledMetadata.Has(id) && !force {
		return nil
	}

	src, err := kvTheoMetadata.Get(id)
	if err != nil {
		return err
	}

	defer src.Close()

	return kvOsLangInstalledMetadata.Set(id, src)
}
