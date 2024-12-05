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
	"golang.org/x/exp/slices"
	"net/url"
	"strings"
)

type GitHubReleaseSelector struct {
	Owner string
	Repo  string
	Tags  []string
	All   bool
}

func CacheGitHubReleasesHandler(u *url.URL) error {

	q := u.Query()

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	releaseSelector := releaseSelectorFromUrl(u)
	force := q.Has("force")

	return CacheGitHubReleases(operatingSystems, releaseSelector, force)
}

func CacheGitHubReleases(operatingSystems []vangogh_local_data.OperatingSystem, releaseSelector *GitHubReleaseSelector, force bool) error {

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

	for _, os := range operatingSystems {
		for _, repo := range data.AllGitHubSources() {

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

			selectedReleases := selectReleases(&repo, releases, releaseSelector)

			if len(selectedReleases) == 0 {
				continue
			}

			if err := cacheRepoReleases(&repo, selectedReleases, dc, force); err != nil {
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

	asset := selectAsset(ghs, release)
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

func selectAsset(ghs *data.GitHubSource, release *github_integration.GitHubRelease) *github_integration.GitHubAsset {

	if len(release.Assets) == 1 {
		return &release.Assets[0]
	}

	filteredAssets := make([]github_integration.GitHubAsset, 0, len(release.Assets))

	for _, asset := range release.Assets {
		skipAsset := false
		for _, exc := range ghs.AssetExclude {
			if exc != "" && strings.Contains(asset.Name, exc) {
				skipAsset = true
			}
		}
		if skipAsset {
			continue
		}
		filteredAssets = append(filteredAssets, asset)
	}

	if len(filteredAssets) == 1 {
		return &filteredAssets[0]
	}

	for _, asset := range filteredAssets {
		for _, inc := range ghs.AssetInclude {
			if inc != "" && strings.Contains(asset.Name, inc) {
				return &asset
			}
		}
	}

	return nil

}

func releaseSelectorFromUrl(u *url.URL) *GitHubReleaseSelector {
	q := u.Query()

	if q.Has(data.GitHubOwnerProperty) ||
		q.Has(data.GitHubRepoProperty) ||
		q.Has(data.GitHubTagProperty) ||
		q.Has(data.GitHubAllReleasesProperty) {

		ghss := &GitHubReleaseSelector{
			Owner: q.Get(data.GitHubOwnerProperty),
			Repo:  q.Get(data.GitHubRepoProperty),
			All:   q.Has(data.GitHubAllReleasesProperty),
		}

		if q.Has(data.GitHubTagProperty) {
			ghss.Tags = strings.Split(q.Get(data.GitHubTagProperty), ",")
		}

		return ghss

	}

	return nil
}

func selectReleases(ghs *data.GitHubSource, releases []github_integration.GitHubRelease, selector *GitHubReleaseSelector) []github_integration.GitHubRelease {
	if selector == nil {
		if len(releases) > 0 {
			return []github_integration.GitHubRelease{releases[0]}
		}
		return releases
	}

	if selector.Owner != "" && ghs.Owner != selector.Owner {
		return nil
	}

	if selector.Repo != "" && ghs.Repo != selector.Repo {
		return nil
	}

	if len(selector.Tags) == 0 {
		if selector.All {
			return releases
		} else if len(releases) > 0 {
			return []github_integration.GitHubRelease{releases[0]}
		}
	}

	var taggedReleases []github_integration.GitHubRelease

	for _, rel := range releases {
		if slices.Contains(selector.Tags, rel.TagName) {
			taggedReleases = append(taggedReleases, rel)
		}
	}

	return taggedReleases
}
