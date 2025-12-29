package cli

import (
	"net/url"

	"github.com/arelate/theo/data"
	"github.com/boggydigital/backups"
	"github.com/boggydigital/nod"
)

func BackupMetadataHandler(_ *url.URL) error {
	return BackupMetadata()
}

func BackupMetadata() error {

	ba := nod.NewProgress("backing up local metadata...")
	defer ba.Done()

	backupsDir := data.Pwd.AbsDirPath(data.Backups)
	metadataDir := data.Pwd.AbsDirPath(data.Metadata)

	if err := backups.Compress(metadataDir, backupsDir); err != nil {
		return err
	}

	ca := nod.NewProgress("cleaning up old backups...")
	defer ca.Done()

	if err := backups.Cleanup(backupsDir, true, ca); err != nil {
		return err
	}

	return nil
}
