package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
)

func MigrateHandler(u *url.URL) error {
	return Migrate()
}

func Migrate() error {

	ma := nod.NewProgress("migrating kevlar data...")
	defer ma.EndWithResult("done")

	dirs := make([]string, 0)

	if theoMetadataDir, err := pathways.GetAbsRelDir(data.TheoMetadata); err == nil {
		dirs = append(dirs, theoMetadataDir)
	} else {
		return ma.EndWithError(err)
	}

	if installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata); err == nil {
		for _, operatingSystem := range vangogh_integration.AllOperatingSystems() {
			osInstalledMetadataDir := filepath.Join(installedMetadataDir, data.OsLangCode(operatingSystem, defaultLangCode))
			if _, err = os.Stat(osInstalledMetadataDir); err == nil {
				dirs = append(dirs, osInstalledMetadataDir)
			}
		}
	} else {
		return ma.EndWithError(err)
	}

	if githubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases); err == nil {
		dirs = append(dirs, githubReleasesDir)
	} else {
		return ma.EndWithError(err)
	}

	if reduxDir, err := pathways.GetAbsRelDir(data.Redux); err == nil {
		dirs = append(dirs, reduxDir)
	} else {
		return ma.EndWithError(err)
	}

	ma.TotalInt(len(dirs))

	for _, dir := range dirs {

		if err := kevlar.Migrate(dir); err != nil {
			return ma.EndWithError(err)
		}
		ma.Increment()
	}

	return nil
}
