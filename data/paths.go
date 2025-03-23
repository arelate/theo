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

const inventoryExt = ".txt"

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
	Redux              pathways.RelDir = "_redux"
	DownloadsManifests pathways.RelDir = "downloads-manifests"
	InstalledManifests pathways.RelDir = "installed-manifests"
	MacOsExtracts      pathways.RelDir = "_macos_extracts"
	GitHubReleases     pathways.RelDir = "github-releases"
	Assets             pathways.RelDir = "assets"
	Binaries           pathways.RelDir = "binaries"
	PrefixArchive      pathways.RelDir = "prefix-archive"
	UmuConfigs         pathways.RelDir = "umu-configs"
	Inventory          pathways.RelDir = "_inventory"
)

var RelToAbsDirs = map[pathways.RelDir]pathways.AbsDir{
	Redux:              Metadata,
	DownloadsManifests: Metadata,
	InstalledManifests: Metadata,
	GitHubReleases:     Metadata,
	Assets:             Runtimes,
	Binaries:           Runtimes,
	PrefixArchive:      Backups,
	MacOsExtracts:      Downloads,
	UmuConfigs:         Runtimes,
	Inventory:          InstalledApps,
}

var AllAbsDirs = []pathways.AbsDir{
	Backups,
	Metadata,
	Downloads,
	Runtimes,
	InstalledApps,
}

func GetAbsBinariesDir(repo string, release *github_integration.GitHubRelease) (string, error) {
	binariesDir, err := pathways.GetAbsRelDir(Binaries)
	if err != nil {
		return "", err
	}

	return filepath.Join(binariesDir, repo, busan.Sanitize(release.TagName)), nil
}

func GetAbsReleasesDir(repo string, release *github_integration.GitHubRelease) (string, error) {
	assetsDir, err := pathways.GetAbsRelDir(Assets)
	if err != nil {
		return "", err
	}

	return filepath.Join(assetsDir, repo, busan.Sanitize(release.TagName)), nil
}

func GetAbsReleaseAssetPath(repo string, release *github_integration.GitHubRelease, asset *github_integration.GitHubAsset) (string, error) {
	relDir, err := GetAbsReleasesDir(repo, release)
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

func GetAbsInventoryFilename(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {
	if err := rdx.MustHave(SlugProperty); err != nil {
		return "", err
	}

	inventoryDir, err := pathways.GetAbsRelDir(Inventory)
	if err != nil {
		return "", err
	}

	osLangInventoryDir := filepath.Join(inventoryDir, OsLangCode(operatingSystem, langCode))

	if slug, ok := rdx.GetLastVal(SlugProperty, id); ok && slug != "" {
		return filepath.Join(osLangInventoryDir, slug+inventoryExt), nil
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

func GetAbsBundlePath(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {

	installedAppsDir, err := pathways.GetAbsDir(InstalledApps)
	if err != nil {
		return "", err
	}

	osLangInstalledAppsDir := filepath.Join(installedAppsDir, OsLangCode(operatingSystem, langCode))

	var bundleProperty string

	switch operatingSystem {
	case vangogh_integration.MacOS:
		bundleProperty = BundleNameProperty
	case vangogh_integration.Linux:
		bundleProperty = SlugProperty
	case vangogh_integration.Windows:
		return "", errors.New("support for Windows bundle path is not implemented")
	default:
		return "", errors.New("unsupported operating system: " + operatingSystem.String())
	}

	if err = rdx.MustHave(bundleProperty); err != nil {
		return "", err
	}

	if appBundle, ok := rdx.GetLastVal(bundleProperty, id); ok && appBundle != "" {
		return filepath.Join(osLangInstalledAppsDir, appBundle), nil
	}

	return "", errors.New(bundleProperty + " is not defined for product " + id)
}
