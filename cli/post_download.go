package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os/exec"
	"path/filepath"
)

func PostDownloadHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, _ := OsLangCodeDownloadType(u)

	return PostDownload(ids, operatingSystems, langCodes)
}

func PostDownload(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string) error {

	pda := nod.NewProgress("performing post-download actions...")
	defer pda.EndWithResult("done")

	vangogh_local_data.PrintParams(ids, operatingSystems, nil, nil, true)

	pda.TotalInt(len(ids))

	installerDownloadType := []vangogh_local_data.DownloadType{vangogh_local_data.Installer}

	for _, id := range ids {

		if metadata, err := GetDownloadMetadata(id, operatingSystems, langCodes, installerDownloadType, false); err == nil {

			for _, link := range metadata.DownloadLinks {

				linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
				if linkOs == vangogh_local_data.Linux {
					if err := linuxPostDownloadActions(id, &link); err != nil {
						return pda.EndWithError(err)
					}
				}
			}

		} else {
			return pda.EndWithError(err)
		}

		pda.Increment()
	}

	return nil
}

func linuxPostDownloadActions(id string, link *vangogh_local_data.DownloadLink) error {

	lpda := nod.Begin(" performing Linux post-download actions for %s...", id)
	defer lpda.EndWithResult("done")

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return lpda.EndWithError(err)
	}

	productInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

	return chmodExecutable(productInstallerPath)
}

func chmodExecutable(path string) error {

	// chmod +x path/to/file
	cmd := exec.Command("chmod", "+x", path)
	return cmd.Run()
}
