package cli

import (
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func downloadGitHubLatestRelease(operatingSystem vangogh_integration.OperatingSystem, force bool) error {

	cra := nod.Begin(" caching GitHub releases for %s...", operatingSystem)
	defer cra.Done()

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return err
	}

	kvGitHubReleases, err := kevlar.New(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	for _, repo := range vangogh_integration.OperatingSystemGitHubRepos(operatingSystem) {

		latestRelease, err := github_integration.GetLatestRelease(repo, kvGitHubReleases)
		if err != nil {
			return err
		}

		if latestRelease == nil {
			continue
		}

		if err = downloadRepoRelease(repo, latestRelease, dc, force); err != nil {
			return err
		}
	}

	return nil
}

func downloadRepoRelease(repo string, release *github_integration.GitHubRelease, dc *dolo.Client, force bool) error {

	crra := nod.Begin(" - tag: %s...", release.TagName)
	defer crra.Done()

	asset := github_integration.GetReleaseAsset(repo, release)
	if asset == nil {
		crra.EndWithResult("asset not found")
		return nil
	}

	ru, err := url.Parse(asset.BrowserDownloadUrl)
	if err != nil {
		return err
	}

	relDir, err := data.GetAbsReleasesDir(repo, release)
	if err != nil {
		return err
	}

	dra := nod.NewProgress(" - asset: %s", asset.Name)
	defer dra.Done()

	if err = dc.Download(ru, force, dra, relDir); err != nil {
		return err
	}

	return nil
}
