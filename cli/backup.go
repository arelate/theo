package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/backups"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func BackupHandler(_ *url.URL) error {
	return Backup()
}

func Backup() error {

	ba := nod.NewProgress("backing up local data...")
	defer ba.End()

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

	ba.EndWithResult("done")

	ca := nod.NewProgress("cleaning up old backups...")
	defer ca.End()

	if err := backups.Cleanup(backupsDir, true, ca); err != nil {
		return ca.EndWithError(err)
	}

	ca.EndWithResult("done")

	return nil
}
