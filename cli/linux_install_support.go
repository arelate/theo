package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"os"
	"os/exec"
	"path/filepath"
)

func linuxPostDownloadActions(id string, link *vangogh_local_data.DownloadLink) error {

	lpda := nod.Begin(" performing Linux post-download actions for %s...", id)
	defer lpda.EndWithResult("done")

	if data.CurrentOS() != vangogh_local_data.Linux {
		return lpda.EndWithError(errors.New("Linux post-download actions are only supported on Linux"))
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return lpda.EndWithError(err)
	}

	productInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

	return chmodExecutable(productInstallerPath)
}

func chmodExecutable(path string) error {

	cea := nod.Begin(" setting executable attribute...")
	defer cea.EndWithResult("done")

	// chmod +x path/to/file
	cmd := exec.Command("chmod", "+x", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
