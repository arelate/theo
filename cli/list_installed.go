package cli

import (
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

	downloadsMetadataDir, err := pathways.GetAbsRelDir(data.DownloadsMetadata)
	if err != nil {
		return lia.EndWithError(err)
	}

	kvDownloadsMetadata, err := kevlar.NewKeyValues(downloadsMetadataDir, kevlar.JsonExt)
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

	ids, err := kvDownloadsMetadata.Keys()
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
		if title != "" {
			summary[id] = append(summary[id], title)
		}
		if bundleName != "" {
			summary[id] = append(summary[id], filepath.Join(installationDir, bundleName))
		}
	}

	lia.EndWithSummary("installed:", summary)

	return nil
}
