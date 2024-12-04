package data

import (
	"github.com/arelate/vangogh_local_data"
	"path"
)

type GitHubRepository struct {
	Owner          string
	Repo           string
	Description    string
	AssetSelectors []string
}

func (ghr *GitHubRepository) String() string {
	return path.Join(ghr.Owner, ghr.Repo)
}

var MacOsWineStaging = GitHubRepository{
	//https://github.com/Gcenx/macOS_Wine_builds
	Owner:          "Gcenx",
	Repo:           "macOS_Wine_builds",
	Description:    "Official Winehq macOS Packages",
	AssetSelectors: []string{"wine-stable", "wine-staging"},
}

var MacOsDxVk = GitHubRepository{
	//https://github.com/Gcenx/DXVK-macOS
	Owner:       "Gcenx",
	Repo:        "DXVK-macOS",
	Description: "Vulkan-based implementation of D3D10 and D3D11 for macOS / Wine",
}

var MacOsGamePortingToolkit = GitHubRepository{
	//https://github.com/Gcenx/game-porting-toolkit
	Owner:       "Gcenx",
	Repo:        "game-porting-toolkit",
	Description: "Apple's Game Porting Toolkit",
}

var LinuxGeProton = GitHubRepository{
	//https://github.com/GloriousEggroll/proton-ge-custom
	Owner:          "GloriousEggroll",
	Repo:           "proton-ge-custom",
	Description:    "Compatibility tool for Steam Play based on Wine and additional components",
	AssetSelectors: []string{".tar.gz"},
}

var OperatingSystemRepos = map[vangogh_local_data.OperatingSystem][]GitHubRepository{
	vangogh_local_data.MacOS: {MacOsWineStaging, MacOsDxVk, MacOsGamePortingToolkit},
	vangogh_local_data.Linux: {LinuxGeProton},
}
