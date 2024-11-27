package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
)

func RemoveExtractsHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, _ := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return RemoveExtracts(ids, operatingSystems, langCodes, force)
}

func RemoveExtracts(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	force bool) error {

	rea := nod.NewProgress("removing extracts...")
	defer rea.End()

	vangogh_local_data.PrintParams(ids, operatingSystems, langCodes, nil, true)

	rea.TotalInt(len(ids))

	extractsDir, err := pathways.GetAbsDir(data.Extracts)
	if err != nil {
		return rea.EndWithError(err)
	}

	for _, id := range ids {

		if title, links, err := GetTitleDownloadLinks(id, operatingSystems, langCodes, nil, force); err == nil {
			if err = removeProductExtracts(id, title, extractsDir, links); err != nil {
				return rea.EndWithError(err)
			}
		} else {
			return rea.EndWithError(err)
		}

		rea.Increment()
	}

	rea.EndWithResult("done")

	return nil
}

func removeProductExtracts(id, title string,
	extractsDir string,
	downloadLinks []vangogh_local_data.DownloadLink) error {

	rela := nod.Begin(" removing extracts for %s...", title)
	defer rela.End()

	idPath := filepath.Join(extractsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rela.EndWithResult("product extracts dir not present")
		return nil
	}

	for _, dl := range downloadLinks {

		path := filepath.Join(extractsDir, id, dl.LocalFilename)

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fa.EndWithResult("not present")
			continue
		}

		if err := os.RemoveAll(path); err != nil {
			return fa.EndWithError(err)
		}

		fa.EndWithResult("done")
	}

	if entries, err := os.ReadDir(idPath); err == nil && len(entries) == 0 {
		rdda := nod.Begin(" removing empty product extracts directory...")
		if err := os.Remove(idPath); err != nil {
			return rdda.EndWithError(err)
		}
		rdda.EndWithResult("done")
	} else if err == nil && hasOnlyDSStore(entries) {
		rdda := nod.Begin(" removing product extracts directory with .DS_Store...")
		if err := os.RemoveAll(idPath); err != nil {
			return rdda.EndWithError(err)
		}
		rdda.EndWithResult("done")
	} else if err != nil {
		return rela.EndWithError(err)
	}

	rela.EndWithResult("done")

	return nil
}

func hasOnlyDSStore(entries []fs.DirEntry) bool {
	if len(entries) == 1 {
		return entries[0].Name() == ".DS_Store"
	}
	return false
}
