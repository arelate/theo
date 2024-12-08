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

	ids := Ids(u)

	return RevealDownloads(ids)
}

func RevealDownloads(ids []string) error {

	rda := nod.Begin("revealing downloads...")
	defer rda.EndWithResult("done")

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return rda.EndWithError(err)
	}

	if len(ids) == 0 {
		return revealCurrentOs(downloadsDir)
	}

	for _, id := range ids {
		if err := revealCurrentOs(filepath.Join(downloadsDir, id)); err != nil {
			return rda.EndWithError(err)
		}
	}

	return nil
}

func revealCurrentOs(path string) error {

	var err error

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

	return err
}

func revealMacOs(path string) error {
	cmd := exec.Command("open", "-R", path)
	return cmd.Run()
}

func revealWindows(path string) error {
	return errors.New("support for reveal on Windows is not implemented")
}

func revealLinux(path string) error {
	cmd := exec.Command("xdg-open", path)
	return cmd.Run()
}
