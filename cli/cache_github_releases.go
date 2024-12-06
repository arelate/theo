package cli

import (
	"encoding/json"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func CacheGitHubReleasesHandler(u *url.URL) error {

	q := u.Query()

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	releaseSelector := data.ReleaseSelectorFromUrl(u)
	force := q.Has("force")

	return CacheGitHubReleases(operatingSystems, releaseSelector, force)
}

func CacheGitHubReleases(operatingSystems []vangogh_local_data.OperatingSystem, releaseSelector *data.GitHubReleaseSelector, force bool) error {

	cra := nod.Begin("caching GitHub releases...")
	defer cra.EndWithResult("done")

	PrintReleaseSelector(operatingSystems, releaseSelector)

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return cra.EndWithError(err)
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return cra.EndWithError(err)
	}

	dc := dolo.DefaultClient

	githubSources, err := data.AllGitHubSources()
	if err != nil {
		return cra.EndWithError(err)
	}

	for _, os := range operatingSystems {
		for _, repo := range githubSources {

			if repo.OS != os {
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

			selectedReleases := data.SelectReleases(repo, releases, releaseSelector)

			if len(selectedReleases) == 0 {
				continue
			}

			if err := cacheRepoReleases(repo, selectedReleases, dc, force); err != nil {
				return cra.EndWithError(err)
			}
		}
	}

	return nil
}

func cacheRepoReleases(ghs *data.GitHubSource, releases []github_integration.GitHubRelease, dc *dolo.Client, force bool) error {

	crrsa := nod.Begin(" %s...", ghs.String())
	defer crrsa.EndWithResult("cached requested releases")

	for _, rel := range releases {

		if err := cacheRepoRelease(ghs, &rel, dc, force); err != nil {
			return crrsa.EndWithError(err)
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
