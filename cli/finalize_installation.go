package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os/exec"
	"path/filepath"
)

func FinalizeInstallationHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, _ := OsLangCodeDownloadType(u)

	return FinalizeInstallation(ids, operatingSystems, langCodes)
}

func FinalizeInstallation(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string) error {

	fia := nod.NewProgress("finalizing installation...")
	defer fia.End()

	PrintParams(ids, operatingSystems, nil, nil)

	fia.TotalInt(len(ids))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return fia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties)
	if err != nil {
		return fia.EndWithError(err)
	}

	installationDir := defaultInstallationDir
	if setupInstallDir, ok := rdx.GetLastVal(data.SetupProperties, data.InstallationPathProperty); ok && setupInstallDir != "" {
		installationDir = setupInstallDir
	}

	installerDownloadType := []vangogh_local_data.DownloadType{vangogh_local_data.Installer}

	for _, id := range ids {

		if title, links, err := GetTitleDownloadLinks(id, operatingSystems, langCodes, installerDownloadType, false); err == nil {
			if err = finalizeProductInstallation(id, title, links, installationDir); err != nil {
				return fia.EndWithError(err)
			}
		} else {
			return fia.EndWithError(err)
		}

		fia.Increment()
	}

	fia.EndWithResult("done")

	return nil
}

func finalizeProductInstallation(id, title string, links []vangogh_local_data.DownloadLink, installationDir string) error {
	fpia := nod.NewProgress(" finalizing installation for %s...", title)
	defer fpia.End()

	extractsDir, err := pathways.GetAbsDir(data.Extracts)
	if err != nil {
		return fpia.EndWithError(err)
	}

	productExtractsDir := filepath.Join(extractsDir, id)

	for _, link := range links {

		linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
		if linkOs != vangogh_local_data.MacOS {
			// currently only macOS finalization is supported (required?)
			continue
		}

		absPostInstallScriptPath := PostInstallScriptPath(productExtractsDir, link)

		pis, err := ParsePostInstallScript(absPostInstallScriptPath)
		if err != nil {
			return fpia.EndWithError(err)
		}

		bundleName := pis.BundleName()

		bundlePath := filepath.Join(installationDir, bundleName)

		if err := removeXattrs(bundlePath); err != nil {
			return err
		}

	}

	fpia.EndWithResult("done")
	return nil
}

func removeXattrs(path string) error {

	// xattr -c -r /Applications/Bundle Name.app
	cmd := exec.Command("xattr", "-c", "-r", path)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
