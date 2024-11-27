package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

func RevealInstalledHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, _ := OsLangCodeDownloadType(u)

	return RevealInstalled(ids, operatingSystems, langCodes)
}

func RevealInstalled(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string) error {

	fia := nod.NewProgress("revealing installed products...")
	defer fia.EndWithResult("done")

	vangogh_local_data.PrintParams(ids, operatingSystems, nil, nil, true)

	fia.TotalInt(len(ids))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return fia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties, data.BundleNameProperty)
	if err != nil {
		return fia.EndWithError(err)
	}

	installationDir := defaultInstallationDir
	if setupInstallDir, ok := rdx.GetLastVal(data.SetupProperties, data.InstallationPathProperty); ok && setupInstallDir != "" {
		installationDir = setupInstallDir
	}

	if len(ids) == 0 {
		return revealCurrentOs(installationDir)
	}

	for _, id := range ids {

		bundleName, _ := rdx.GetLastVal(data.BundleNameProperty, id)

		installedPath := filepath.Join(installationDir, bundleName)
		if err := revealCurrentOs(installedPath); err != nil {
			return fia.EndWithError(err)
		}

		fia.Increment()
	}

	return nil
}

func revealProductInstallation(title string, link vangogh_local_data.DownloadLink) error {
	rpia := nod.Begin(" revealing %s...", title)
	defer rpia.End()

	rpia.EndWithResult("done")

	return nil
}
