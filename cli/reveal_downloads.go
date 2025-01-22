package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

func RevealDownloadsHandler(u *url.URL) error {

	ids := Ids(u)

	return RevealDownloads(ids...)
}

func RevealDownloads(ids ...string) error {

	rda := nod.Begin("revealing downloads...")
	defer rda.EndWithResult("done")

	return currentOsRevealDownloads(ids...)
}

func currentOsRevealDownloads(ids ...string) error {

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	// only reveal a single product downloads directory, otherwise, reveal all downloads dir
	if len(ids) == 1 {
		productDownloadsDir := filepath.Join(downloadsDir, ids[0])
		return currentOsReveal(productDownloadsDir)
	} else {
		return currentOsReveal(downloadsDir)
	}
}

func currentOsReveal(path string) error {
	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		return macOsReveal(path)
	case vangogh_integration.Windows:
		return windowsReveal(path)
	case vangogh_integration.Linux:
		return linuxReveal(path)
	default:
		return errors.New("cannot reveal on unknown operating system")
	}
}
