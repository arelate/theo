package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func RevealBackupsHandler(_ *url.URL) error {
	return RevealBackups()
}

func RevealBackups() error {

	rda := nod.Begin("revealing backups...")
	defer rda.EndWithResult("done")

	backupsDir, err := pathways.GetAbsDir(data.Backups)
	if err != nil {
		return err
	}

	return currentOsReveal(backupsDir)
}
