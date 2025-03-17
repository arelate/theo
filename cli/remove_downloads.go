package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"os"
	"path/filepath"
)

func RemoveDownloadsHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return RemoveDownloads(operatingSystems, langCodes, downloadTypes, rdx, force, ids...)
}

func RemoveDownloads(operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	rdx redux.Writeable,
	force bool,
	ids ...string) error {

	rda := nod.NewProgress("removing downloads...")
	defer rda.Done()

	vangogh_integration.PrintParams(ids, operatingSystems, langCodes, downloadTypes, true)

	rda.TotalInt(len(ids))

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	for _, id := range ids {

		metadata, err := getTheoMetadata(id, rdx, force)
		if err != nil {
			return err
		}

		if err = removeProductDownloadLinks(id, metadata, operatingSystems, langCodes, downloadTypes, downloadsDir); err != nil {
			return err
		}

		rda.Increment()
	}

	return nil
}

func removeProductDownloadLinks(id string,
	metadata *vangogh_integration.TheoMetadata,
	operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	downloadsDir string) error {

	rdla := nod.Begin(" removing downloads for %s...", metadata.Title)
	defer rdla.Done()

	idPath := filepath.Join(downloadsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rdla.EndWithResult("product downloads dir not present")
		return nil
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(operatingSystems...).
		FilterLanguageCodes(langCodes...).
		FilterDownloadTypes(downloadTypes...)

	if len(dls) == 0 {
		rdla.EndWithResult("no links are matching operating params")
		return nil
	}

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
			return err
		}

		fa.Done()
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)
	if entries, err := os.ReadDir(productDownloadsDir); err == nil && len(entries) == 0 {
		rdda := nod.Begin(" removing empty product downloads directory...")
		if err = os.Remove(productDownloadsDir); err != nil {
			return err
		}
		rdda.Done()
	} else {
		return err
	}

	return nil
}
