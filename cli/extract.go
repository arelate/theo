package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	pkgExt = ".pkg"
	exeExt = ".exe"
	shExt  = ".sh"
)

func ExtractHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return Extract(ids, operatingSystems, langCodes, downloadTypes, force)
}

func Extract(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	ea := nod.NewProgress("extracting installers game data...")
	defer ea.EndWithResult("done")

	vangogh_local_data.PrintParams(ids, operatingSystems, langCodes, downloadTypes, true)

	ea.TotalInt(len(ids))

	for _, id := range ids {

		if metadata, err := GetDownloadMetadata(id, operatingSystems, langCodes, downloadTypes, force); err == nil {
			if err = extractProductDownloadLinks(id, metadata, force); err != nil {
				return ea.EndWithError(err)
			}
		} else {
			return ea.EndWithError(err)
		}

		ea.Increment()
	}

	return nil
}

func extractProductDownloadLinks(id string, metadata *vangogh_local_data.DownloadMetadata, force bool) error {

	epdla := nod.NewProgress(" extracting %s, please wait...", metadata.Title)
	defer epdla.EndWithResult("done")

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return epdla.EndWithError(err)
	}

	extractsDir, err := pathways.GetAbsDir(data.Extracts)
	if err != nil {
		return epdla.EndWithError(err)
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)
	productExtractsDir := filepath.Join(extractsDir, id)

	if _, err := os.Stat(productExtractsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(productExtractsDir, 0755); err != nil {
			return epdla.EndWithError(err)
		}
	}

	for _, link := range metadata.DownloadLinks {

		linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
		linkExt := filepath.Ext(link.LocalFilename)

		if linkOs == vangogh_local_data.MacOS && linkExt == pkgExt {
			if err := extractMacOsInstaller(link, productDownloadsDir, productExtractsDir, force); err != nil {
				return epdla.EndWithError(err)
			}
		}
		if linkOs == vangogh_local_data.Windows && linkExt == exeExt {
			if err := extractWindowsInstaller(link, productDownloadsDir, productExtractsDir, force); err != nil {
				return epdla.EndWithError(err)
			}
		}
	}

	return nil
}

func extractMacOsInstaller(link vangogh_local_data.DownloadLink, productDownloadsDir, productExtractsDir string, force bool) error {

	if CurrentOS() != vangogh_local_data.MacOS {
		return errors.New("extracting .pkg installers is only supported on macOS")
	}

	localFilenameExtractsDir := filepath.Join(productExtractsDir, link.LocalFilename)
	// if the product extracts dir already exists - that would imply that the product
	// has been extracted already. Remove the directory with contents if forced
	// Return early otherwise (if not forced).
	if _, err := os.Stat(localFilenameExtractsDir); err == nil {
		if force {
			if err := os.RemoveAll(localFilenameExtractsDir); err != nil {
				return err
			}
		} else {
			return nil
		}
	}

	localDownload := filepath.Join(productDownloadsDir, link.LocalFilename)

	cmd := exec.Command("pkgutil", "--expand-full", localDownload, localFilenameExtractsDir)

	return cmd.Run()
}

func extractWindowsInstaller(link vangogh_local_data.DownloadLink, downloadsDir, extractsDir string, force bool) error {
	return nil
}
