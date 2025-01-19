package data

import (
	_ "embed"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"strings"
)

type GitHubSource struct {
	OwnerRepo string
	Asset     string
}

var geProtonCustom = &GitHubSource{
	OwnerRepo: "GloriousEggroll/proton-ge-custom",
	Asset:     ".tar.gz",
}

var umuLauncher = &GitHubSource{
	OwnerRepo: "Open-Wine-Components/umu-launcher",
	Asset:     "Zipapp.zip",
}

func (ghs *GitHubSource) GetAsset(release *github_integration.GitHubRelease) *github_integration.GitHubAsset {

	if len(release.Assets) == 1 {
		return &release.Assets[0]
	}

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, ghs.Asset) {
			return &asset
		}
	}

	return nil
}

func OsGitHubSources(os vangogh_integration.OperatingSystem) []*GitHubSource {
	switch os {
	case vangogh_integration.Linux:
		return []*GitHubSource{geProtonCustom, umuLauncher}
	default:
		return nil
	}
}
