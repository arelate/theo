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

func pinInstalledMetadata(operatingSystems []vangogh_integration.OperatingSystem, force bool, ids ...string) error {

	pima := nod.NewProgress("pinning theo metadata as installed...")
	defer pima.EndWithResult("done")

	theoMetadataDir, err := pathways.GetAbsRelDir(data.TheoMetadata)
	if err != nil {
		return pima.EndWithError(err)
	}

	kvTheoMetadata, err := kevlar.NewKeyValues(theoMetadataDir, kevlar.JsonExt)
	if err != nil {
		return pima.EndWithError(err)
	}

	pima.TotalInt(len(ids))

	for _, id := range ids {

		if err := pinTheoMetadata(id, operatingSystems, kvTheoMetadata, force); err != nil {
			return pima.EndWithError(err)
		}

		pima.Increment()

	}

	return nil
}

func pinTheoMetadata(id string,
	operatingSystems []vangogh_integration.OperatingSystem,
	kvTheoMetadata kevlar.KeyValues,
	force bool) error {

	for _, os := range operatingSystems {
		if err := pinInstalledMetadataForOs(id, os, kvTheoMetadata, force); err != nil {
			return err
		}
	}

	return nil
}

func pinInstalledMetadataForOs(id string,
	os vangogh_integration.OperatingSystem,
	kvTheoMetadata kevlar.KeyValues,
	force bool) error {

	pimoa := nod.Begin(" pinning metadata as installed for %s on %s...", id, os)
	defer pimoa.EndWithResult("done")

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return pimoa.EndWithError(err)
	}

	osInstalledMetadataDir := filepath.Join(installedMetadataDir, os.String())

	kvOsInstalledMetadata, err := kevlar.NewKeyValues(osInstalledMetadataDir, kevlar.JsonExt)
	if err != nil {
		return pimoa.EndWithError(err)
	}

	hasTheoMetadata, err := kvTheoMetadata.Has(id)
	if err != nil {
		return pimoa.EndWithError(err)
	}

	if !hasTheoMetadata {
		return errors.New("theo metadata not found for: " + id)
	}

	hasInstalledMetadata, err := kvOsInstalledMetadata.Has(id)
	if err != nil {
		return pimoa.EndWithError(err)
	}

	if hasInstalledMetadata && !force {
		return nil
	}

	src, err := kvTheoMetadata.Get(id)
	if err != nil {
		return pimoa.EndWithError(err)
	}

	defer src.Close()

	return kvOsInstalledMetadata.Set(id, src)
}
