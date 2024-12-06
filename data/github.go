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
	"path"
	"strings"
)

type GitHubSource struct {
	OS           vangogh_local_data.OperatingSystem
	Owner        string
	Repo         string
	Description  string
	AssetInclude []string
	AssetExclude []string
}

type WineGitHubSource struct {
	*GitHubSource
	BinaryPath string
	Default    bool
}

type GitHubReleaseSelector struct {
	Owner string
	Repo  string
	Tags  []string
	All   bool
}

func (ghs *GitHubSource) String() string {
	return path.Join(ghs.Owner, ghs.Repo)
}

var MacOsWineStaging = &WineGitHubSource{
	//https://github.com/Gcenx/macOS_Wine_builds
	GitHubSource: &GitHubSource{
		OS:           vangogh_local_data.MacOS,
		Owner:        "Gcenx",
		Repo:         "macOS_Wine_builds",
		Description:  "Official Winehq macOS Packages",
		AssetInclude: []string{"wine-stable", "wine-staging"},
	},
	BinaryPath: "Wine Staging.app/Contents/Resources/wine/bin/wine",
}

var MacOsDxVk = GitHubSource{
	//https://github.com/Gcenx/DXVK-macOS
	OS:           vangogh_local_data.MacOS,
	Owner:        "Gcenx",
	Repo:         "DXVK-macOS",
	Description:  "Vulkan-based implementation of D3D10 and D3D11 for macOS / Wine",
	AssetExclude: []string{"CrossOver", "crossover", "async"},
}

var MacOsGamePortingToolkit = &WineGitHubSource{
	//https://github.com/Gcenx/game-porting-toolkit
	GitHubSource: &GitHubSource{
		OS:          vangogh_local_data.MacOS,
		Owner:       "Gcenx",
		Repo:        "game-porting-toolkit",
		Description: "Apple's Game Porting Toolkit",
	},
	BinaryPath: "Game Porting Toolkit.app/Contents/Resources/wine/bin/wine64",
	Default:    true,
}

var LinuxGeProton = &WineGitHubSource{
	//https://github.com/GloriousEggroll/proton-ge-custom
	GitHubSource: &GitHubSource{
		OS:           vangogh_local_data.Linux,
		Owner:        "GloriousEggroll",
		Repo:         "proton-ge-custom",
		Description:  "Compatibility tool for Steam Play based on Wine and additional components",
		AssetInclude: []string{".tar.gz"},
	},
	Default: true,
}

func AllGitHubSources() ([]*GitHubSource, error) {
	return []*GitHubSource{
		MacOsWineStaging.GitHubSource,
		&MacOsDxVk,
		MacOsGamePortingToolkit.GitHubSource,
		LinuxGeProton.GitHubSource,
	}, nil
}

func AllWineSources() ([]*WineGitHubSource, error) {
	return []*WineGitHubSource{
		MacOsWineStaging,
		MacOsGamePortingToolkit,
		LinuxGeProton,
	}, nil
}

func GetWineSource(os vangogh_local_data.OperatingSystem, owner, repo string) (*WineGitHubSource, error) {

	wineSources, err := AllWineSources()
	if err != nil {
		return nil, err
	}

	for _, ws := range wineSources {
		if ws.OS == os &&
			ws.Owner == owner &&
			ws.Repo == repo {
			return ws, nil
		}
	}
	return nil, errors.New("WINE source not found")
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

func GetDefaultWineSource(os vangogh_local_data.OperatingSystem) (*WineGitHubSource, error) {
	wineSources, err := AllWineSources()
	if err != nil {
		return nil, err
	}

	for _, ws := range wineSources {
		if ws.OS == os &&
			ws.Default {
			return ws, nil
		}
	}

	return nil, errors.New("cannot determine default WINE source for " + os.String())
}
