package cli

import (
	"fmt"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

func ListInstalledHandler(u *url.URL) error {
	return ListInstalled()
}

func ListInstalled() error {

	lia := nod.Begin("listing installed products...")
	defer lia.EndWithResult("done")

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return lia.EndWithError(err)
	}

	kvInstalledMetadata, err := kevlar.NewKeyValues(installedMetadataDir, kevlar.JsonExt)
	if err != nil {
		return lia.EndWithError(err)
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return lia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir,
		data.SetupProperties,
		data.TitleProperty,
		data.BundleNameProperty)
	if err != nil {
		return lia.EndWithError(err)
	}

	installationDir := defaultInstallationDir
	if setupInstallDir, ok := rdx.GetLastVal(data.SetupProperties, data.InstallationPathProperty); ok && setupInstallDir != "" {
		installationDir = setupInstallDir
	}

	ids, err := kvInstalledMetadata.Keys()
	if err != nil {
		return lia.EndWithError(err)
	}

	summary := make(map[string][]string)

	for _, id := range ids {

		title, bundleName := "", ""
		if pt, ok := rdx.GetLastVal(data.TitleProperty, id); ok {
			title = pt
		}
		if bn, ok := rdx.GetLastVal(data.BundleNameProperty, id); ok {
			bundleName = bn
		}
		if title != "" && bundleName != "" {
			heading := fmt.Sprintf("%s (%s)", title, id)
			bundlePath := filepath.Join(installationDir, bundleName)
			summary[heading] = append(summary[heading], bundlePath)
		}
	}

	lia.EndWithSummary("installed:", summary)

	return nil
}
