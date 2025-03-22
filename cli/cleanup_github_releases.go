package cli

import (
	"encoding/json"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"iter"
	"maps"
	"os"
	"path/filepath"
)

func cleanupGitHubReleases(os vangogh_integration.OperatingSystem) error {

	cra := nod.Begin("cleaning up cached GitHub releases, keeping the latest for %s...", os)
	defer cra.Done()

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return err
	}

	kvGitHubReleases, err := kevlar.New(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	for _, repo := range github_integration.OsGitHubSources(os) {

		if err = cleanupRepoReleases(repo, kvGitHubReleases); err != nil {
			return err
		}
	}

	return nil
}

func cleanupRepoReleases(ghs *github_integration.GitHubSource, kvGitHubReleases kevlar.KeyValues) error {
	crra := nod.Begin(" %s...", ghs.OwnerRepo)
	defer crra.Done()

	rcReleases, err := kvGitHubReleases.Get(ghs.OwnerRepo)
	if err != nil {
		return err
	}
	defer rcReleases.Close()

	var releases []github_integration.GitHubRelease
	if err = json.NewDecoder(rcReleases).Decode(&releases); err != nil {
		return err
	}

	cleanupFiles := make([]string, 0)

	for ii, release := range releases {
		if ii == 0 {
			continue
		}

		asset := ghs.GetAsset(&release)
		if asset == nil {
			continue
		}

		absReleaseAssetPath, err := data.GetAbsReleaseAssetPath(ghs, &release, asset)
		if err != nil {
			return err
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
			return err
		}
	}

	return nil
}

func removeRepoReleasesFiles(absFilePaths []string) error {
	rfa := nod.NewProgress("cleaning up older releases files...")
	defer rfa.Done()

	rfa.TotalInt(len(absFilePaths))

	absDirs := make(map[string]any)

	for _, absFilePath := range absFilePaths {
		dir, _ := filepath.Split(absFilePath)
		absDirs[dir] = nil
		if err := os.Remove(absFilePath); err != nil {
			return err
		}

		rfa.Increment()
	}

	return removeRepoReleaseDirs(maps.Keys(absDirs))
}

func removeRepoReleaseDirs(absDirs iter.Seq[string]) error {
	rda := nod.Begin("cleaning up older releases directories...")
	defer rda.Done()

	for absDir := range absDirs {
		if err := removeDirIfEmpty(absDir); err != nil {
			return err
		}
	}
	return nil
}
