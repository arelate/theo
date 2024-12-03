package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
)

const (
	relPayloadPath = "package.pkg/Scripts/payload"

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

	vangogh_local_data.PrintParams(ids, operatingSystems, langCodes, downloadTypes, true)

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

		if metadata, err := GetDownloadMetadata(id, operatingSystems, langCodes, downloadTypes, force); err == nil {
			if err = placeExtractedProductDownloadLinks(id, metadata, installationDir, force); err != nil {
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

func placeExtractedProductDownloadLinks(id string, metadata *vangogh_local_data.DownloadMetadata, installationDir string, force bool) error {

	pedla := nod.NewProgress(" placing extracted data for %s...", metadata.Title)
	defer pedla.End()

	extractsDir, err := pathways.GetAbsDir(data.Extracts)
	if err != nil {
		return pedla.EndWithError(err)
	}

	productExtractsDir := filepath.Join(extractsDir, id)

	for _, link := range metadata.DownloadLinks {

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
		if linkOs == vangogh_local_data.Linux && linkExt == shExt {
			if err := placeLinuxExtracts(link, productExtractsDir, installationDir, force); err != nil {
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

	absPostInstallScriptPath := PostInstallScriptPath(productExtractsDir, link)
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

	installerType := postInstallScript.InstallerType()

	absInstallationPath := filepath.Join(installationDir, bundleName)

	switch installerType {
	case "game":
		return placeMacOsGame(absExtractPayloadPath, absInstallationPath, force)
	case "dlc":
		return placeMacOsDlc(absExtractPayloadPath, absInstallationPath, force)
	default:
		return errors.New("unknown postinstall script installer type: " + installerType)
	}
}

func placeMacOsGame(absExtractsPayloadPath, absInstallationPath string, force bool) error {

	// when installing a game
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

	return os.Rename(absExtractsPayloadPath, absInstallationPath)
}

func placeMacOsDlc(absExtractsPayloadPath, absInstallationPath string, force bool) error {

	if _, err := os.Stat(absInstallationPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absInstallationPath, 0755); err != nil {
			return err
		}
	}

	// enumerate all DLC files in the payload directory
	dlcFiles := make([]string, 0)

	if err := filepath.Walk(absExtractsPayloadPath, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if relPath, err := filepath.Rel(absExtractsPayloadPath, path); err == nil {
				dlcFiles = append(dlcFiles, relPath)
			} else {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	for _, dlcFile := range dlcFiles {

		absDstPath := filepath.Join(absInstallationPath, dlcFile)
		absDstDir, _ := filepath.Split(absDstPath)

		if _, err := os.Stat(absDstDir); os.IsNotExist(err) {
			if err := os.MkdirAll(absDstDir, 0755); err != nil {
				return err
			}
		}

		absSrcPath := filepath.Join(absExtractsPayloadPath, dlcFile)

		if err := os.Rename(absSrcPath, absDstPath); err != nil {
			return err
		}
	}

	return nil
}

func placeWindowsExtracts(link vangogh_local_data.DownloadLink, productExtractsDir, installationDir string, force bool) error {
	return errors.New("placing Windows extracts is not implemented")
}

func placeLinuxExtracts(link vangogh_local_data.DownloadLink, productExtractsDir, installationDir string, force bool) error {
	return errors.New("placing Linux extracts is not implemented")
}
