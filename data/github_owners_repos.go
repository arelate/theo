package data

import (
	"github.com/arelate/vangogh_local_data"
	"path"
)

type GitHubSource struct {
	OS           vangogh_local_data.OperatingSystem
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
	OS:           vangogh_local_data.MacOS,
	Owner:        "Gcenx",
	Repo:         "macOS_Wine_builds",
	Description:  "Official Winehq macOS Packages",
	AssetInclude: []string{"wine-stable", "wine-staging"},
}

var MacOsDxVk = GitHubSource{
	//https://github.com/Gcenx/DXVK-macOS
	OS:           vangogh_local_data.MacOS,
	Owner:        "Gcenx",
	Repo:         "DXVK-macOS",
	Description:  "Vulkan-based implementation of D3D10 and D3D11 for macOS / Wine",
	AssetExclude: []string{"CrossOver", "crossover", "async"},
}

var MacOsGamePortingToolkit = GitHubSource{
	//https://github.com/Gcenx/game-porting-toolkit
	OS:          vangogh_local_data.MacOS,
	Owner:       "Gcenx",
	Repo:        "game-porting-toolkit",
	Description: "Apple's Game Porting Toolkit",
}

var LinuxGeProton = GitHubSource{
	//https://github.com/GloriousEggroll/proton-ge-custom
	OS:           vangogh_local_data.Linux,
	Owner:        "GloriousEggroll",
	Repo:         "proton-ge-custom",
	Description:  "Compatibility tool for Steam Play based on Wine and additional components",
	AssetInclude: []string{".tar.gz"},
}

func AllGitHubSources() []GitHubSource {
	return []GitHubSource{
		MacOsWineStaging,
		MacOsDxVk,
		MacOsGamePortingToolkit,
		LinuxGeProton,
	}
}
