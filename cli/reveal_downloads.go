package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os/exec"
	"path/filepath"
)

func RevealDownloadsHandler(u *url.URL) error {
	return RevealDownloads(u.Query().Get("id"))
}

func RevealDownloads(id string) error {

	rda := nod.Begin("revealing downloads for %s...", id)
	defer rda.End()

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return rda.EndWithError(err)
	}

	path := filepath.Join(downloadsDir, id)

	switch CurrentOS() {
	case vangogh_local_data.MacOS:
		err = revealMacOs(path)
	case vangogh_local_data.Windows:
		err = revealWindows(path)
	case vangogh_local_data.Linux:
		err = revealLinux(path)
	default:
		err = errors.New("cannot reveal on unknown operating system")
	}

	if err != nil {
		return rda.EndWithError(err)
	}

	rda.EndWithResult("done")

	return nil
}

func revealMacOs(path string) error {
	cmd := exec.Command("open", "-R", path)
	return cmd.Run()
}

func revealWindows(path string) error {
	return errors.New("support for reveal on Windows is not implemented")
}

func revealLinux(path string) error {
	return errors.New("support for reveal on Linux is not implemented")
}
