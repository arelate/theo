package data

import (
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const theoDirname = "theo"

const manifestExt = ".txt"

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
	Manifests         pathways.RelDir = "_manifests"
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
	Manifests:         InstalledApps,
}

var AllAbsDirs = []pathways.AbsDir{
	Backups,
	Metadata,
	Downloads,
	Runtimes,
	InstalledApps,
	//Prefixes,
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

func GetPrefixName(id string, rdx redux.Readable) (string, error) {
	if slug, ok := rdx.GetLastVal(SlugProperty, id); ok && slug != "" {
		return slug, nil
	} else {
		return "", errors.New("product slug is undefined: " + id)
	}
}

func OsLangCode(operatingSystem vangogh_integration.OperatingSystem, langCode string) string {
	return strings.Join([]string{operatingSystem.String(), langCode}, "-")
}

func GetAbsPrefixDir(id, langCode string, rdx redux.Readable) (string, error) {
	if err := rdx.MustHave(SlugProperty); err != nil {
		return "", err
	}

	installedAppsDir, err := pathways.GetAbsDir(InstalledApps)
	if err != nil {
		return "", err
	}

	osLangInstalledAppsDir := filepath.Join(installedAppsDir, OsLangCode(vangogh_integration.Windows, langCode))

	prefixName, err := GetPrefixName(id, rdx)
	if err != nil {
		return "", err
	}

	return filepath.Join(osLangInstalledAppsDir, prefixName), nil
}

func GetAbsManifestFilename(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {
	if err := rdx.MustHave(SlugProperty); err != nil {
		return "", err
	}

	manifestsDir, err := pathways.GetAbsRelDir(Manifests)
	if err != nil {
		return "", err
	}

	osLangManifestsDir := filepath.Join(manifestsDir, OsLangCode(operatingSystem, langCode))

	if slug, ok := rdx.GetLastVal(SlugProperty, id); ok && slug != "" {
		return filepath.Join(osLangManifestsDir, slug+manifestExt), nil
	} else {
		return "", errors.New("product slug is undefined: " + id)
	}
}

func GetRelFilesModifiedAfter(absDir string, utcTime int64) ([]string, error) {
	files := make([]string, 0)

	if err := filepath.Walk(absDir, func(path string, info fs.FileInfo, err error) error {

		if err != nil {
			return err
		}

		//if info.IsDir() {
		//	return nil
		//}

		if info.ModTime().UTC().Unix() >= utcTime {
			relPath, err := filepath.Rel(absDir, path)
			if err != nil {
				return err
			}

			files = append(files, relPath)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}
