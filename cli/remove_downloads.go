package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
)

func RemoveDownloadsHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, _ := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return RemoveDownloads(ids, operatingSystems, langCodes, force)
}

func RemoveDownloads(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	force bool) error {

	rda := nod.NewProgress("removing downloads...")
	defer rda.End()

	PrintParams(ids, operatingSystems, langCodes, nil)

	rda.TotalInt(len(ids))

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return rda.EndWithError(err)
	}

	for _, id := range ids {

		if title, links, err := GetTitleDownloadLinks(id, operatingSystems, langCodes, nil, force); err == nil {
			if err = removeProductDownloadLinks(id, title, downloadsDir, links); err != nil {
				return rda.EndWithError(err)
			}
		} else {
			return rda.EndWithError(err)
		}

		rda.Increment()
	}

	rda.EndWithResult("done")

	return nil
}

func removeProductDownloadLinks(id, title string,
	downloadsDir string,
	downloadLinks []vangogh_local_data.DownloadLink) error {

	rdla := nod.Begin(" removing downloads for %s...", title)
	defer rdla.End()

	idPath := filepath.Join(downloadsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rdla.EndWithResult("product downloads dir not present")
		return nil
	}

	for _, dl := range downloadLinks {

		path := filepath.Join(downloadsDir, id, dl.LocalFilename)

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fa.EndWithResult("not present")
			continue
		}

		if err := os.Remove(path); err != nil {
			return fa.EndWithError(err)
		}

		fa.EndWithResult("done")
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)
	if entries, err := os.ReadDir(productDownloadsDir); err == nil && len(entries) == 0 {
		rdda := nod.Begin(" removing empty product downloads directory...")
		if err := os.Remove(productDownloadsDir); err != nil {
			return rdda.EndWithError(err)
		}
		rdda.EndWithResult("done")
	} else {
		return rdla.EndWithError(err)
	}

	rdla.EndWithResult("done")

	return nil
}
