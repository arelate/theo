package data

import (
	"encoding/json"
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/pathways"
	"strings"
)

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

func GetWineSourceLatestRelease(wineRepo string) (*WineGitHubSource, *github_integration.GitHubRelease, error) {

	gitHubReleasesDir, err := pathways.GetAbsRelDir(GitHubReleases)
	if err != nil {
		return nil, nil, err
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return nil, nil, err
	}

	wineSource, err := GetWineSource(wineRepo)
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

	var latestRelease *github_integration.GitHubRelease
	if len(releases) > 0 {
		latestRelease = &releases[0]
	}

	if latestRelease == nil {
		return nil, nil, errors.New("nil WINE releases match selector")
	}

	return wineSource, latestRelease, nil
}

func GetDxVkSourceLatestRelease(dxVkRepo string) (*GitHubSource, *github_integration.GitHubRelease, error) {
	gitHubReleasesDir, err := pathways.GetAbsRelDir(GitHubReleases)
	if err != nil {
		return nil, nil, err
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return nil, nil, err
	}

	dxVkSource, err := GetDxVkSource(dxVkRepo)
	if err != nil {
		return nil, nil, err
	}

	rcReleases, err := kvGitHubReleases.Get(dxVkSource.String())
	if err != nil {
		return nil, nil, err
	}
	defer rcReleases.Close()

	var releases []github_integration.GitHubRelease
	if err := json.NewDecoder(rcReleases).Decode(&releases); err != nil {
		return nil, nil, err
	}

	var latestRelease *github_integration.GitHubRelease
	if len(releases) > 0 {
		latestRelease = &releases[0]
	}

	if latestRelease == nil {
		return nil, nil, errors.New("nil DXVK releases match selector")
	}

	return dxVkSource, latestRelease, nil
}
