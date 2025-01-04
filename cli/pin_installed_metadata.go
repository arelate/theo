package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func PinInstalledMetadataHandler(u *url.URL) error {

	ids := Ids(u)
	force := u.Query().Has("force")

	return PinInstalledMetadata(ids, force)
}

func PinInstalledMetadata(ids []string, force bool) error {

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

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return pima.EndWithError(err)
	}

	kvInstalledMetadata, err := kevlar.NewKeyValues(installedMetadataDir, kevlar.JsonExt)
	if err != nil {
		return pima.EndWithError(err)
	}

	pima.TotalInt(len(ids))

	for _, id := range ids {

		if err := pinTheoMetadata(id, kvTheoMetadata, kvInstalledMetadata, force); err != nil {
			return pima.EndWithError(err)
		}

		pima.Increment()

	}

	return nil
}

func pinTheoMetadata(id string, kvTheoMetadata, kvInstalledMetadata kevlar.KeyValues, force bool) error {

	hasTheoMetadata, err := kvTheoMetadata.Has(id)
	if err != nil {
		return err
	}

	if !hasTheoMetadata {
		return errors.New("theo metadata not found for: " + id)
	}

	hasInstalledMetadata, err := kvInstalledMetadata.Has(id)
	if err != nil {
		return err
	}

	if hasInstalledMetadata && !force {
		return nil
	}

	src, err := kvTheoMetadata.Get(id)
	if err != nil {
		return err
	}

	defer src.Close()

	return kvInstalledMetadata.Set(id, src)
}
