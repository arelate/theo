package cli

import (
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"path/filepath"
)

func downloadGitHubLatestRelease(operatingSystem vangogh_integration.OperatingSystem, since int64, force bool) error {

	cra := nod.Begin(" downloading GitHub latest releases for %s...", operatingSystem)
	defer cra.Done()

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return err
	}

	kvGitHubReleases, err := kevlar.New(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if force {
		since = -1
	}

	updatedReleases := kvGitHubReleases.Since(since, kevlar.Create, kevlar.Update)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	for repo := range updatedReleases {

		latestRelease, err := github_integration.GetLatestRelease(repo, kvGitHubReleases)
		if err != nil {
			return err
		}

		if latestRelease == nil {
			continue
		}

		if err = downloadRepoLatestRelease(repo, latestRelease, rdx, dc, force); err != nil {
			return err
		}
	}

	return nil
}

func downloadRepoLatestRelease(repo string, release *github_integration.GitHubRelease, rdx redux.Readable, dc *dolo.Client, force bool) error {

	crra := nod.Begin(" - tag: %s...", release.TagName)
	defer crra.Done()

	asset := github_integration.GetReleaseAsset(repo, release)
	if asset == nil {
		crra.EndWithResult("asset not found")
		return nil
	}

	glau, err := data.ServerUrl(rdx, data.ServerGitHubLatestAssetPath, map[string]string{"repo": repo})
	if err != nil {
		return err
	}

	relDir, err := data.GetAbsReleasesDir(repo, release)
	if err != nil {
		return err
	}

	dra := nod.NewProgress(" - asset: %s", asset.Name)
	defer dra.Done()

	_, assetFilename := filepath.Split(asset.BrowserDownloadUrl)

	if err = dc.Download(glau, force, dra, relDir, assetFilename); err != nil {
		return err
	}

	return nil
}
