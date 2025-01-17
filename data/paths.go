package data

import (
	"fmt"
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
	Wine          pathways.AbsDir = "wine"
	GitHubSources pathways.AbsDir = "github-sources"
	InstalledApps pathways.AbsDir = "installed-apps"
	Prefixes      pathways.AbsDir = "prefixes"
)

const (
	Redux             pathways.RelDir = "_redux"
	TheoMetadata      pathways.RelDir = "theo"
	InstalledMetadata pathways.RelDir = "installed"
	MacOsExtracts     pathways.RelDir = "_macos_extracts"
	GitHubReleases    pathways.RelDir = "github-releases"
	Releases          pathways.RelDir = "releases"
	Binaries          pathways.RelDir = "binaries"
	PrefixArchive     pathways.RelDir = "prefix-archive"
)

var RelToAbsDirs = map[pathways.RelDir]pathways.AbsDir{
	Redux:             Metadata,
	TheoMetadata:      Metadata,
	InstalledMetadata: Metadata,
	GitHubReleases:    Metadata,
	Releases:          Wine,
	Binaries:          Wine,
	PrefixArchive:     Backups,
	MacOsExtracts:     Downloads,
}

var AllAbsDirs = []pathways.AbsDir{
	Backups,
	Metadata,
	Downloads,
	Wine,
	GitHubSources,
	InstalledApps,
	Prefixes,
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

//func GetPrefixName(id, langCode string, rdx kevlar.ReadableRedux) (string, error) {
//	if err := rdx.MustHave(SlugProperty); err != nil {
//		return "", err
//	}
//
//	if slug, ok := rdx.GetLastVal(SlugProperty, id); ok && slug != "" {
//		return fmt.Sprintf("%s-%s", slug, langCode), nil
//	}
//
//	return "", nil
//}

func GetAbsPrefixDir(id, langCode string) (string, error) {
	prefixesDir, err := pathways.GetAbsDir(Prefixes)
	if err != nil {
		return "", err
	}

	return path.Join(prefixesDir, fmt.Sprintf("%s-%s", id, langCode)), nil
}

func OsLangCodeDir(os vangogh_integration.OperatingSystem, langCode string) string {
	return fmt.Sprintf("%s-%s", os.String(), langCode)
}
