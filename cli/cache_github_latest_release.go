package cli

import (
	"encoding/json"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func cacheGitHubLatestRelease(force bool) error {

	currentOs := data.CurrentOS()

	cra := nod.Begin(" caching GitHub releases for %s...", currentOs)
	defer cra.EndWithResult("done")

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return cra.EndWithError(err)
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return cra.EndWithError(err)
	}

	dc := dolo.DefaultClient

	githubSources, err := data.LoadGitHubSources()
	if err != nil {
		return cra.EndWithError(err)
	}

	for _, repo := range githubSources {

		if repo.OS != data.CurrentOS() {
			continue
		}

		rcReleases, err := kvGitHubReleases.Get(repo.String())
		if err != nil {
			return cra.EndWithError(err)
		}

		var releases []github_integration.GitHubRelease
		if err := json.NewDecoder(rcReleases).Decode(&releases); err != nil {
			rcReleases.Close()
			return cra.EndWithError(err)
		}

		if err := rcReleases.Close(); err != nil {
			return cra.EndWithError(err)
		}

		var latestRelease *github_integration.GitHubRelease
		if len(releases) > 0 {
			latestRelease = &releases[0]
		}

		if latestRelease == nil {
			continue
		}

		if err := cacheRepoRelease(repo, latestRelease, dc, force); err != nil {
			return cra.EndWithError(err)
		}
	}

	return nil
}

func cacheRepoRelease(ghs *data.GitHubSource, release *github_integration.GitHubRelease, dc *dolo.Client, force bool) error {

	crra := nod.Begin(" - tag: %s...", release.TagName)
	defer crra.EndWithResult("done")

	asset := data.SelectAsset(ghs, release)
	if asset == nil {
		crra.EndWithResult("asset not found")
		return nil
	}

	ru, err := url.Parse(asset.BrowserDownloadUrl)
	if err != nil {
		return crra.EndWithError(err)
	}

	relDir, err := data.GetAbsReleasesDir(ghs, release)
	if err != nil {
		return crra.EndWithError(err)
	}

	dra := nod.NewProgress(" - asset: %s", asset.Name)
	defer dra.EndWithResult("done")

	if err := dc.Download(ru, force, dra, relDir); err != nil {
		return crra.EndWithError(err)
	}

	return nil
}
