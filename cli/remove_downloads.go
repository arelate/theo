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
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return RemoveDownloads(ids, operatingSystems, langCodes, downloadTypes, force)
}

func RemoveDownloads(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	rda := nod.NewProgress("removing downloads...")
	defer rda.EndWithResult("done")

	vangogh_local_data.PrintParams(ids, operatingSystems, langCodes, downloadTypes, true)

	rda.TotalInt(len(ids))

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return rda.EndWithError(err)
	}

	for _, id := range ids {

		metadata, err := getTheoMetadata(id, force)
		if err != nil {
			return rda.EndWithError(err)
		}

		if err = removeProductDownloadLinks(id, metadata, operatingSystems, langCodes, downloadTypes, downloadsDir); err != nil {
			return rda.EndWithError(err)
		}

		rda.Increment()
	}

	return nil
}

func removeProductDownloadLinks(id string,
	metadata *vangogh_local_data.TheoMetadata,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	downloadsDir string) error {

	rdla := nod.Begin(" removing downloads for %s...", metadata.Title)
	defer rdla.EndWithResult("done")

	idPath := filepath.Join(downloadsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rdla.EndWithResult("product downloads dir not present")
		return nil
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(operatingSystems...).
		FilterLanguageCodes(langCodes...).
		FilterDownloadTypes(downloadTypes...)

	for _, dl := range dls {
		// if we don't do this - product downloads dir itself will be removed
		if dl.LocalFilename == "" {
			continue
		}

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

	return nil
}
