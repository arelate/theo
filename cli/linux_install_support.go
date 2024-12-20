package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"os/exec"
	"path/filepath"
)

func linuxPostDownloadActions(id string, link *vangogh_local_data.DownloadLink) error {

	if CurrentOS() != vangogh_local_data.Linux {
		return errors.New("Linux post-download actions are only supported on Linux")
	}

	lpda := nod.Begin(" performing Linux post-download actions for %s...", id)
	defer lpda.EndWithResult("done")

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return lpda.EndWithError(err)
	}

	productInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

	return chmodExecutable(productInstallerPath)
}

func chmodExecutable(path string) error {

	// chmod +x path/to/file
	cmd := exec.Command("chmod", "+x", path)
	return cmd.Run()
}
