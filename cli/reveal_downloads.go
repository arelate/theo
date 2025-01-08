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

	return currentOsRevealDownloads(ids...)
}

func currentOsRevealDownloads(ids ...string) error {

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return currentOsReveal(downloadsDir)
	}

	for _, id := range ids {
		productDownloadsDir := filepath.Join(downloadsDir, id)
		if err := currentOsReveal(productDownloadsDir); err != nil {
			return err
		}
	}

	return nil
}

func currentOsReveal(path string) error {
	switch data.CurrentOS() {
	case vangogh_local_data.MacOS:
		return macOsReveal(path)
	case vangogh_local_data.Windows:
		return windowsReveal(path)
	case vangogh_local_data.Linux:
		return linuxReveal(path)
	default:
		return errors.New("cannot reveal on unknown operating system")
	}
}

func macOsReveal(path string) error {
	cmd := exec.Command("open", "-R", path)
	return cmd.Run()
}

func windowsReveal(path string) error {
	return errors.New("support for reveal on Windows is not implemented")
}

func linuxReveal(path string) error {
	cmd := exec.Command("xdg-open", path)
	return cmd.Run()
}
