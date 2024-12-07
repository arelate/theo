package data

import (
	"encoding/json"
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
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

func SelectAsset(ghs *GitHubSource, release *github_integration.GitHubRelease) *github_integration.GitHubAsset {

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

func ReleaseSelectorFromUrl(u *url.URL) *GitHubReleaseSelector {
	q := u.Query()

	if q.Has(GitHubOwnerProperty) ||
		q.Has(GitHubRepoProperty) ||
		q.Has(GitHubTagProperty) ||
		q.Has(GitHubAllReleasesProperty) {

		ghss := &GitHubReleaseSelector{
			Owner: q.Get(GitHubOwnerProperty),
			Repo:  q.Get(GitHubRepoProperty),
			All:   q.Has(GitHubAllReleasesProperty),
		}

		if q.Has(GitHubTagProperty) {
			ghss.Tags = strings.Split(q.Get(GitHubTagProperty), ",")
		}

		return ghss

	}

	return nil
}

func SelectReleases(ghs *GitHubSource, releases []github_integration.GitHubRelease, selector *GitHubReleaseSelector) []github_integration.GitHubRelease {
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

func GetWineSourceRelease(os vangogh_local_data.OperatingSystem, releaseSelector *GitHubReleaseSelector) (*WineGitHubSource, *github_integration.GitHubRelease, error) {

	gitHubReleasesDir, err := pathways.GetAbsRelDir(GitHubReleases)
	if err != nil {
		return nil, nil, err
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return nil, nil, err
	}

	wineSource, err := GetWineSource(os, releaseSelector.Owner, releaseSelector.Repo)
	if err != nil {
		return nil, nil, err
	}

	rcReleases, err := kvGitHubReleases.Get(wineSource.String())
	if err != nil {
		return nil, nil, err
	}
	defer rcReleases.Close()

	var releases []github_integration.GitHubRelease
	if err := json.NewDecoder(rcReleases).Decode(&releases); err != nil {
		return nil, nil, err
	}

	selRels := SelectReleases(wineSource.GitHubSource, releases, releaseSelector)

	if selRels == nil {
		return nil, nil, errors.New("nil releases match selector")
	} else if len(selRels) == 0 {
		return nil, nil, errors.New("no releases match selector")
	} else if len(selRels) > 1 {
		return nil, nil, errors.New("multiple releases match selector")
	}

	return wineSource, &selRels[0], nil
}
