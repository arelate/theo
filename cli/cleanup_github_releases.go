package cli

import (
	"encoding/json"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"golang.org/x/exp/maps"
	"net/url"
	"os"
	"path/filepath"
)

func CleanupGitHubReleasesHandler(u *url.URL) error {
	operatingSystems, _, _ := OsLangCodeDownloadType(u)

	return CleanupGitHubReleases(operatingSystems)
}

func CleanupGitHubReleases(operatingSystems []vangogh_local_data.OperatingSystem) error {

	cra := nod.Begin("cleaning up cached GitHub releases, keeping the latest...")
	defer cra.EndWithResult("done")

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return cra.EndWithError(err)
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return cra.EndWithError(err)
	}

	for _, os := range operatingSystems {
		for _, repo := range data.AllGitHubSources() {

			if repo.OS != os {
				continue
			}

			if err := cleanupRepoReleases(&repo, kvGitHubReleases); err != nil {
				return cra.EndWithError(err)
			}
		}
	}

	return nil
}

func cleanupRepoReleases(ghs *data.GitHubSource, kvGitHubReleases kevlar.KeyValues) error {
	crra := nod.Begin(" %s...", ghs.String())
	defer crra.EndWithResult("done")

	rcReleases, err := kvGitHubReleases.Get(ghs.String())
	if err != nil {
		return crra.EndWithError(err)
	}
	defer rcReleases.Close()

	var releases []github_integration.GitHubRelease
	if err := json.NewDecoder(rcReleases).Decode(&releases); err != nil {
		return crra.EndWithError(err)
	}

	cleanupFiles := make([]string, 0)

	for ii, release := range releases {
		if ii == 0 {
			continue
		}

		asset := data.SelectAsset(ghs, &release)
		if asset == nil {
			continue
		}

		absReleaseAssetPath, err := data.GetAbsReleaseAssetPath(ghs, &release, asset)
		if err != nil {
			return crra.EndWithError(err)
		}

		if _, err := os.Stat(absReleaseAssetPath); err == nil {
			cleanupFiles = append(cleanupFiles, absReleaseAssetPath)
		}
	}

	if len(cleanupFiles) == 0 {
		crra.EndWithResult("already clean")
		return nil
	} else {
		if err := removeRepoReleasesFiles(cleanupFiles); err != nil {
			return crra.EndWithError(err)
		}
	}

	return nil
}

func removeRepoReleasesFiles(absFilePaths []string) error {
	rfa := nod.NewProgress("cleaning up older releases files...")
	defer rfa.EndWithResult("done")

	rfa.TotalInt(len(absFilePaths))

	absDirs := make(map[string]any)

	for _, absFilePath := range absFilePaths {
		dir, _ := filepath.Split(absFilePath)
		absDirs[dir] = nil
		if err := os.Remove(absFilePath); err != nil {
			return rfa.EndWithError(err)
		}

		rfa.Increment()
	}

	return removeRepoReleaseDirs(maps.Keys(absDirs))
}

func removeRepoReleaseDirs(absDirs []string) error {
	rda := nod.NewProgress("cleaning up older releases directories...")
	defer rda.EndWithResult("done")

	rda.TotalInt(len(absDirs))

	for _, absDir := range absDirs {
		if err := removeDirIfEmpty(absDir); err != nil {
			return rda.EndWithError(err)
		}

		rda.Increment()
	}
	return nil
}
