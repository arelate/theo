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

	pima := nod.NewProgress("pinning metadata as installed...")
	defer pima.End()

	downloadsMetadataDir, err := pathways.GetAbsRelDir(data.DownloadsMetadata)
	if err != nil {
		return pima.EndWithError(err)
	}

	kvDownloadsMetadata, err := kevlar.NewKeyValues(downloadsMetadataDir, kevlar.JsonExt)
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

		if err := pinDownloadsMetadata(id, kvDownloadsMetadata, kvInstalledMetadata, force); err != nil {
			return pima.EndWithError(err)
		}

		pima.Increment()

	}

	pima.EndWithResult("done")

	return nil
}

func pinDownloadsMetadata(id string, kvDownloadsMetadata, kvInstalledMetadata kevlar.KeyValues, force bool) error {

	hasDownloadsMetadata, err := kvDownloadsMetadata.Has(id)
	if err != nil {
		return err
	}

	if !hasDownloadsMetadata {
		return errors.New("downloads metadata not found for: " + id)
	}

	hasInstalledMetadata, err := kvInstalledMetadata.Has(id)
	if err != nil {
		return err
	}

	if hasInstalledMetadata && !force {
		return nil
	}

	src, err := kvDownloadsMetadata.Get(id)
	if err != nil {
		return err
	}

	defer src.Close()

	return kvInstalledMetadata.Set(id, src)
}
