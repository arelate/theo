package cli

import (
	"encoding/json"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"golang.org/x/exp/slices"
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

	rda.TotalInt(len(ids))

	dmd, err := pathways.GetAbsRelDir(data.DownloadsMetadata)
	if err != nil {
		return rda.EndWithError(err)
	}

	kvdm, err := kevlar.NewKeyValues(dmd, kevlar.JsonExt)
	if err != nil {
		return rda.EndWithError(err)
	}

	for _, id := range ids {

		if err = removeProductDownloads(id, kvdm, operatingSystems, langCodes, force); err != nil {
			return rda.EndWithError(err)
		}

		rda.Increment()
	}

	rda.EndWithResult("done")

	return nil
}

func removeProductDownloads(id string,
	kv kevlar.KeyValues,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	force bool) error {

	rda := nod.Begin(" loading metadata for %s...", id)
	defer rda.End()

	if has, err := kv.Has(id); err == nil {
		if !has {
			rda.EndWithResult("not present")
			return nil
		}
	} else {
		return rda.EndWithError(err)
	}

	dmrc, err := kv.Get(id)
	if err != nil {
		return rda.EndWithError(err)
	}
	defer dmrc.Close()

	var downloadMetadata vangogh_local_data.DownloadMetadata
	if err = json.NewDecoder(dmrc).Decode(&downloadMetadata); err != nil {
		return rda.EndWithError(err)
	}

	var downloadLinks []vangogh_local_data.DownloadLink
	for _, link := range downloadMetadata.DownloadLinks {
		linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
		linkLangCode := link.LanguageCode

		if slices.Contains(operatingSystems, linkOs) &&
			slices.Contains(langCodes, linkLangCode) {
			downloadLinks = append(downloadLinks, link)
		}
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return rda.EndWithError(err)
	}

	if err = removeProductDownloadLinks(id, downloadMetadata.Title, downloadsDir, downloadLinks, force); err != nil {
		return rda.EndWithError(err)
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)
	if entries, err := os.ReadDir(productDownloadsDir); err == nil && len(entries) == 0 {
		rdda := nod.Begin(" removing empty product downloads directory...")
		if err := os.Remove(productDownloadsDir); err != nil {
			return rdda.EndWithError(err)
		}
		rdda.EndWithResult("done")
	} else {
		return rda.EndWithError(err)
	}

	rda.EndWithResult("done")

	return nil
}

func removeProductDownloadLinks(id, title string,
	downloadsDir string,
	downloadLinks []vangogh_local_data.DownloadLink,
	force bool) error {

	rdla := nod.Begin(" removing downloads for %s...", title)
	defer rdla.End()

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

	return nil
}
