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

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return lia.EndWithError(err)
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

			// TODO: this needs to be OS-specific
			bundlePath := filepath.Join(installedAppsDir, bundleName)
			summary[heading] = append(summary[heading], bundlePath)
		}
	}

	if len(summary) == 0 {
		lia.EndWithResult("found nothing")
	} else {
		lia.EndWithSummary("found the following products:", summary)
	}

	return nil
}
