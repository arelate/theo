package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/backups"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func BackupMetadataHandler(_ *url.URL) error {
	return BackupMetadata()
}

func BackupMetadata() error {

	ba := nod.NewProgress("backing up local metadata...")
	defer ba.EndWithResult("done")

	backupsDir, err := pathways.GetAbsDir(data.Backups)
	if err != nil {
		return ba.EndWithError(err)
	}

	metadataDir, err := pathways.GetAbsDir(data.Metadata)
	if err != nil {
		return ba.EndWithError(err)
	}

	if err := backups.Compress(metadataDir, backupsDir); err != nil {
		return ba.EndWithError(err)
	}

	ca := nod.NewProgress("cleaning up old backups...")
	defer ca.EndWithResult("done")

	if err := backups.Cleanup(backupsDir, true, ca); err != nil {
		return ca.EndWithError(err)
	}

	return nil
}
