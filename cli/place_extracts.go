package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
)

const (
	relPostInstallScriptPath = "package.pkg/Scripts/postinstall"
	relPayloadPath           = "package.pkg/Scripts/payload"

	defaultInstallationDir = "/Applications"
)

func PlaceExtractsHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return PlaceExtracts(ids, operatingSystems, langCodes, downloadTypes, force)
}

func PlaceExtracts(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	pea := nod.NewProgress("placing extracts to the installation directory...")
	defer pea.End()

	PrintParams(ids, operatingSystems, langCodes, downloadTypes)

	pea.TotalInt(len(ids))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return pea.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties)
	if err != nil {
		return pea.EndWithError(err)
	}

	installationDir := defaultInstallationDir
	if setupInstallDir, ok := rdx.GetLastVal(data.SetupProperties, data.InstallationPathProperty); ok && setupInstallDir != "" {
		installationDir = setupInstallDir
	}

	if _, err := os.Stat(installationDir); os.IsNotExist(err) {
		if err := os.MkdirAll(installationDir, 0755); err != nil {
			return pea.EndWithError(err)
		}
	}

	for _, id := range ids {

		if title, links, err := GetTitleDownloadLinks(id, operatingSystems, langCodes, downloadTypes, force); err == nil {
			if err = placeExtractedProductDownloadLinks(id, title, links, installationDir, force); err != nil {
				return pea.EndWithError(err)
			}
		} else {
			return pea.EndWithError(err)
		}

		pea.Increment()
	}

	pea.EndWithResult("done")

	return nil

}

func placeExtractedProductDownloadLinks(id, title string, links []vangogh_local_data.DownloadLink, installationDir string, force bool) error {

	pedla := nod.NewProgress(" placing extracted data for %s...", title)
	defer pedla.End()

	extractsDir, err := pathways.GetAbsDir(data.Extracts)
	if err != nil {
		return pedla.EndWithError(err)
	}

	productExtractsDir := filepath.Join(extractsDir, id)

	for _, link := range links {

		linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
		linkExt := filepath.Ext(link.LocalFilename)

		if linkOs == vangogh_local_data.MacOS && linkExt == pkgExt {
			if err := placeMacOsExtracts(link, productExtractsDir, installationDir, force); err != nil {
				return pedla.EndWithError(err)
			}
		}
		if linkOs == vangogh_local_data.Windows && linkExt == exeExt {
			if err := placeWindowsExtracts(link, productExtractsDir, installationDir, force); err != nil {
				return pedla.EndWithError(err)
			}
		}
	}

	pedla.EndWithResult("done")

	return nil
}

func placeMacOsExtracts(link vangogh_local_data.DownloadLink, productExtractsDir, installationDir string, force bool) error {

	if CurrentOS() != vangogh_local_data.MacOS {
		return errors.New("placing .pkg extracts is only supported on macOS")
	}

	localFilenameExtractsDir := filepath.Join(productExtractsDir, link.LocalFilename)
	absPostInstallScriptPath := filepath.Join(localFilenameExtractsDir, relPostInstallScriptPath)

	postInstallScript, err := ParsePostInstallScript(absPostInstallScriptPath)
	if err != nil {
		return err
	}

	absExtractPayloadPath := filepath.Join(productExtractsDir, link.LocalFilename, relPayloadPath)

	if _, err := os.Stat(absExtractPayloadPath); os.IsNotExist(err) {
		return errors.New("cannot locate extracts payload")
	}

	bundleName := postInstallScript.BundleName()
	if bundleName == "" {
		return errors.New("cannot determine bundle name from postinstall file")
	}
	absInstallationPath := filepath.Join(installationDir, bundleName)

	if _, err := os.Stat(absInstallationPath); err == nil {
		if force {
			if err := os.RemoveAll(absInstallationPath); err != nil {
				return err
			}
		} else {
			// already installed, overwrite won't be forced
			return nil
		}
	}

	return os.Rename(absExtractPayloadPath, absInstallationPath)
}

func placeWindowsExtracts(link vangogh_local_data.DownloadLink, productExtractsDir, installationDir string, force bool) error {
	return nil
}
