package data

import (
	"fmt"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/pathways"
	"os"
	"path"
	"path/filepath"
)

const theoDirname = "theo"

func InitRootDir() (string, error) {
	ucd, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	rootDir := filepath.Join(ucd, theoDirname)
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		if err := os.MkdirAll(rootDir, 0755); err != nil {
			return "", err
		}
	}

	for _, ad := range AllAbsDirs {
		absDir := filepath.Join(rootDir, string(ad))
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			if err := os.MkdirAll(absDir, 0755); err != nil {
				return "", err
			}
		}
	}

	for rd, ad := range RelToAbsDirs {
		absRelDir := filepath.Join(rootDir, string(ad), string(rd))
		if _, err := os.Stat(absRelDir); os.IsNotExist(err) {
			if err := os.MkdirAll(absRelDir, 0755); err != nil {
				return "", err
			}
		}
	}

	return filepath.Join(ucd, theoDirname), nil
}

const (
	Backups       pathways.AbsDir = "backups"
	Metadata      pathways.AbsDir = "metadata"
	Downloads     pathways.AbsDir = "downloads"
	Cellars       pathways.AbsDir = "cellars"
	GitHubSources pathways.AbsDir = "github-sources"
	InstalledApps pathways.AbsDir = "installed-apps"
)

const (
	Redux             pathways.RelDir = "_redux"
	DownloadsMetadata pathways.RelDir = "downloads-metadata"
	InstalledMetadata pathways.RelDir = "installed-metadata"
	Extracts          pathways.RelDir = "_extracts"
	GitHubReleases    pathways.RelDir = "github-releases"
	Releases          pathways.RelDir = "rel"
	Binaries          pathways.RelDir = "bin"
	Prefixes          pathways.RelDir = "pfx"
	PrefixArchive     pathways.RelDir = "pfx-archive"
)

var RelToAbsDirs = map[pathways.RelDir]pathways.AbsDir{
	Redux:             Metadata,
	DownloadsMetadata: Metadata,
	InstalledMetadata: Metadata,
	GitHubReleases:    Metadata,
	Releases:          Cellars,
	Binaries:          Cellars,
	Prefixes:          Cellars,
	PrefixArchive:     Backups,
	Extracts:          Downloads,
}

var AllAbsDirs = []pathways.AbsDir{
	Backups,
	Metadata,
	Downloads,
	Cellars,
	GitHubSources,
	InstalledApps,
}

func GetAbsBinariesDir(ghs *GitHubSource, release *github_integration.GitHubRelease) (string, error) {

	binDir, err := pathways.GetAbsRelDir(Binaries)
	if err != nil {
		return "", err
	}

	return filepath.Join(binDir, ghs.String(), busan.Sanitize(release.TagName)), nil
}

func GetAbsReleasesDir(ghs *GitHubSource, release *github_integration.GitHubRelease) (string, error) {

	releasesDir, err := pathways.GetAbsRelDir(Releases)
	if err != nil {
		return "", err
	}

	return filepath.Join(releasesDir, ghs.String(), busan.Sanitize(release.TagName)), nil
}

func GetAbsReleaseAssetPath(ghs *GitHubSource, release *github_integration.GitHubRelease, asset *github_integration.GitHubAsset) (string, error) {
	relDir, err := GetAbsReleasesDir(ghs, release)
	if err != nil {
		return "", err
	}

	_, fn := path.Split(asset.BrowserDownloadUrl)

	return filepath.Join(relDir, fn), nil
}

func GetAbsPrefixDir(name string) (string, error) {
	prefixesDir, err := pathways.GetAbsRelDir(Prefixes)
	if err != nil {
		return "", err
	}

	return filepath.Join(prefixesDir, busan.Sanitize(name)), nil
}

func OsLangCodeDir(os vangogh_local_data.OperatingSystem, langCode string) string {
	return fmt.Sprintf("%s-%s", os.String(), langCode)
}
