package cli

import (
	"fmt"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"io/fs"
	"net/url"
	"path/filepath"
)

func ListInstalledHandler(u *url.URL) error {
	size := u.Query().Has("size")
	langCode := defaultLangCode
	if u.Query().Has(vangogh_local_data.LanguageCodeProperty) {
		langCode = u.Query().Get(vangogh_local_data.LanguageCodeProperty)
	}
	return ListInstalled(langCode, size)
}

func ListInstalled(langCode string, size bool) error {

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

	osLangCodeInstalledAppDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(vangogh_local_data.MacOS, langCode))

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

			bundlePath := filepath.Join(osLangCodeInstalledAppDir, bundleName)
			summary[heading] = append(summary[heading], bundlePath)
			if size {
				if ds, err := dirSize(bundlePath); err == nil {
					summary[heading] = append(summary[heading], fmtBytes(ds))
				}
			}
		}
	}

	if len(summary) == 0 {
		lia.EndWithResult("found nothing")
	} else {
		lia.EndWithSummary("found the following products:", summary)
	}

	return nil
}

func dirSize(path string) (int64, error) {
	var size int64
	if err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		} else if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return -1, err
	}
	return size, nil
}

func fmtBytes(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
