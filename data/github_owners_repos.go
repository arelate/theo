package data

import (
	"github.com/arelate/vangogh_local_data"
	"path"
)

type GitHubSource struct {
	Owner        string
	Repo         string
	Description  string
	AssetInclude []string
	AssetExclude []string
}

func (ghs *GitHubSource) String() string {
	return path.Join(ghs.Owner, ghs.Repo)
}

var MacOsWineStaging = GitHubSource{
	//https://github.com/Gcenx/macOS_Wine_builds
	Owner:        "Gcenx",
	Repo:         "macOS_Wine_builds",
	Description:  "Official Winehq macOS Packages",
	AssetInclude: []string{"wine-stable", "wine-staging"},
}

var MacOsDxVk = GitHubSource{
	//https://github.com/Gcenx/DXVK-macOS
	Owner:        "Gcenx",
	Repo:         "DXVK-macOS",
	Description:  "Vulkan-based implementation of D3D10 and D3D11 for macOS / Wine",
	AssetExclude: []string{"CrossOver", "crossover", "async"},
}

var MacOsGamePortingToolkit = GitHubSource{
	//https://github.com/Gcenx/game-porting-toolkit
	Owner:       "Gcenx",
	Repo:        "game-porting-toolkit",
	Description: "Apple's Game Porting Toolkit",
}

var LinuxGeProton = GitHubSource{
	//https://github.com/GloriousEggroll/proton-ge-custom
	Owner:        "GloriousEggroll",
	Repo:         "proton-ge-custom",
	Description:  "Compatibility tool for Steam Play based on Wine and additional components",
	AssetInclude: []string{".tar.gz"},
}

var OsGitHubSources = map[vangogh_local_data.OperatingSystem][]GitHubSource{
	vangogh_local_data.MacOS: {MacOsWineStaging, MacOsDxVk, MacOsGamePortingToolkit},
	vangogh_local_data.Linux: {LinuxGeProton},
}
