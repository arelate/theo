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
	defer rea.EndWithResult("done")

	vangogh_local_data.PrintParams(ids, operatingSystems, langCodes, nil, true)

	rea.TotalInt(len(ids))

	extractsDir, err := pathways.GetAbsDir(data.Extracts)
	if err != nil {
		return rea.EndWithError(err)
	}

	for _, id := range ids {

		if metadata, err := GetDownloadMetadata(id, operatingSystems, langCodes, nil, force); err == nil {
			if err = removeProductExtracts(id, metadata, extractsDir); err != nil {
				return rea.EndWithError(err)
			}
		} else {
			return rea.EndWithError(err)
		}

		rea.Increment()
	}

	return nil
}

func removeProductExtracts(id string,
	metadata *vangogh_local_data.DownloadMetadata,
	extractsDir string) error {

	rela := nod.Begin(" removing extracts for %s...", metadata.Title)
	defer rela.EndWithResult("done")

	idPath := filepath.Join(extractsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rela.EndWithResult("product extracts dir not present")
		return nil
	}

	for _, dl := range metadata.DownloadLinks {

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

	rdda := nod.Begin(" removing empty product extracts directory...")
	if err := removeDirIfEmpty(idPath); err != nil {
		return rdda.EndWithError(err)
	}
	rdda.EndWithResult("done")

	return nil
}

func hasOnlyDSStore(entries []fs.DirEntry) bool {
	if len(entries) == 1 {
		return entries[0].Name() == ".DS_Store"
	}
	return false
}

func removeDirIfEmpty(dirPath string) error {
	if entries, err := os.ReadDir(dirPath); err == nil && len(entries) == 0 {
		if err := os.Remove(dirPath); err != nil {
			return err
		}
	} else if err == nil && hasOnlyDSStore(entries) {
		if err := os.RemoveAll(dirPath); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
