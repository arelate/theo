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

func cacheGitHubLatestRelease(operatingSystem vangogh_integration.OperatingSystem, force bool) error {

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

	for _, ghs := range github_integration.OsGitHubSources(operatingSystem) {

		latestRelease, err := ghs.GetLatestRelease(kvGitHubReleases)
		if err != nil {
			return err
		}

		if latestRelease == nil {
			continue
		}

		if err = cacheRepoRelease(ghs, latestRelease, dc, force); err != nil {
			return err
		}
	}

	return nil
}

func cacheRepoRelease(ghs *github_integration.GitHubSource, release *github_integration.GitHubRelease, dc *dolo.Client, force bool) error {

	crra := nod.Begin(" - tag: %s...", release.TagName)
	defer crra.Done()

	asset := ghs.GetAsset(release)
	if asset == nil {
		crra.EndWithResult("asset not found")
		return nil
	}

	ru, err := url.Parse(asset.BrowserDownloadUrl)
	if err != nil {
		return err
	}

	relDir, err := data.GetAbsReleasesDir(ghs, release)
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
