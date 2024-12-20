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

	vangogh_local_data.PrintParams(ids, operatingSystems, langCodes, nil, true)

	fia.TotalInt(len(ids))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return fia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties, data.BundleNameProperty)
	if err != nil {
		return fia.EndWithError(err)
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return fia.EndWithError(err)
	}

	if len(ids) == 0 {
		return revealCurrentOs(installedAppsDir)
	}

	for _, id := range ids {

		bundleName, _ := rdx.GetLastVal(data.BundleNameProperty, id)

		// TODO: this needs to be OS-specific
		installedPath := filepath.Join(installedAppsDir, CurrentOS().String(), bundleName)
		if err := revealCurrentOs(installedPath); err != nil {
			return fia.EndWithError(err)
		}

		fia.Increment()
	}

	return nil
}
