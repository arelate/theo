package data

import (
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/pathways"
	"os"
	"path"
	"path/filepath"
)

const theoDirname = "theo"

func InitRootDir() (string, error) {
	udhd, err := UserDataHomeDir()
	if err != nil {
		return "", err
	}

	rootDir := filepath.Join(udhd, theoDirname)
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

	return filepath.Join(udhd, theoDirname), nil
}

const (
	Backups       pathways.AbsDir = "backups"
	Metadata      pathways.AbsDir = "metadata"
	Downloads     pathways.AbsDir = "downloads"
	Runtimes      pathways.AbsDir = "runtimes"
	InstalledApps pathways.AbsDir = "installed-apps"
	Prefixes      pathways.AbsDir = "prefixes"
)

const (
	Redux             pathways.RelDir = "_redux"
	TheoMetadata      pathways.RelDir = "theo"
	InstalledMetadata pathways.RelDir = "installed"
	MacOsExtracts     pathways.RelDir = "_macos_extracts"
	GitHubReleases    pathways.RelDir = "github-releases"
	Assets            pathways.RelDir = "assets"
	Binaries          pathways.RelDir = "binaries"
	PrefixArchive     pathways.RelDir = "prefix-archive"
	UmuConfigs        pathways.RelDir = "umu-configs"
)

var RelToAbsDirs = map[pathways.RelDir]pathways.AbsDir{
	Redux:             Metadata,
	TheoMetadata:      Metadata,
	InstalledMetadata: Metadata,
	GitHubReleases:    Metadata,
	Assets:            Runtimes,
	Binaries:          Runtimes,
	PrefixArchive:     Backups,
	MacOsExtracts:     Downloads,
	UmuConfigs:        Runtimes,
}

var AllAbsDirs = []pathways.AbsDir{
	Backups,
	Metadata,
	Downloads,
	Runtimes,
	InstalledApps,
	Prefixes,
}

func GetAbsBinariesDir(ghs *GitHubSource, release *github_integration.GitHubRelease) (string, error) {
	binariesDir, err := pathways.GetAbsRelDir(Binaries)
	if err != nil {
		return "", err
	}

	return filepath.Join(binariesDir, ghs.OwnerRepo, busan.Sanitize(release.TagName)), nil
}

func GetAbsReleasesDir(ghs *GitHubSource, release *github_integration.GitHubRelease) (string, error) {
	assetsDir, err := pathways.GetAbsRelDir(Assets)
	if err != nil {
		return "", err
	}

	return filepath.Join(assetsDir, ghs.OwnerRepo, busan.Sanitize(release.TagName)), nil
}

func GetAbsReleaseAssetPath(ghs *GitHubSource, release *github_integration.GitHubRelease, asset *github_integration.GitHubAsset) (string, error) {
	relDir, err := GetAbsReleasesDir(ghs, release)
	if err != nil {
		return "", err
	}

	_, fn := path.Split(asset.BrowserDownloadUrl)

	return filepath.Join(relDir, fn), nil
}

func GetPrefixName(id, langCode string) string {
	return id + "-" + langCode
}

func GetAbsPrefixDir(id, langCode string) (string, error) {
	prefixesDir, err := pathways.GetAbsDir(Prefixes)
	if err != nil {
		return "", err
	}

	return path.Join(prefixesDir, GetPrefixName(id, langCode)), nil
}

func OsLangCodeDir(os vangogh_integration.OperatingSystem, langCode string) string {
	return os.String() + "-" + langCode
}
