package cli

import (
	"github.com/arelate/theo/data"
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

	ddp, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return rda.EndWithError(err)
	}

	path := filepath.Join(ddp, id)

	cmd := exec.Command("open", "-R", path)
	if err = cmd.Run(); err != nil {
		return rda.EndWithError(err)
	}

	rda.EndWithResult("done")

	return nil
}
