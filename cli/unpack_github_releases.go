package cli

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/xi2/xz"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	tarXzExt = ".tar.xz"
	tarGzExt = ".tar.gz"
)

func UnpackGitHubReleasesHandler(u *url.URL) error {

	q := u.Query()

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	releaseSelector := releaseSelectorFromUrl(u)
	force := q.Has("force")

	return UnpackGitHubReleases(operatingSystems, releaseSelector, force)
}

func UnpackGitHubReleases(operatingSystems []vangogh_local_data.OperatingSystem, releaseSelector *GitHubReleaseSelector, force bool) error {
	ura := nod.Begin("unpacking GitHub releases...")
	defer ura.EndWithResult("done")

	PrintReleaseSelector(operatingSystems, releaseSelector)

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return ura.EndWithError(err)
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return ura.EndWithError(err)
	}

	for _, os := range operatingSystems {
		for _, repo := range data.AllGitHubSources() {

			if repo.OS != os {
				continue
			}

			rcReleases, err := kvGitHubReleases.Get(repo.String())
			if err != nil {
				return ura.EndWithError(err)
			}

			var releases []github_integration.GitHubRelease
			if err := json.NewDecoder(rcReleases).Decode(&releases); err != nil {
				rcReleases.Close()
				return ura.EndWithError(err)
			}

			if err := rcReleases.Close(); err != nil {
				return ura.EndWithError(err)
			}

			selectedReleases := selectReleases(&repo, releases, releaseSelector)

			if len(selectedReleases) == 0 {
				err = errors.New("no releases selected for unpacking")
				return ura.EndWithError(err)
			}

			if err := unpackRepoReleases(&repo, selectedReleases, force); err != nil {
				return ura.EndWithError(err)
			}
		}
	}

	return nil
}

func unpackRepoReleases(ghs *data.GitHubSource, releases []github_integration.GitHubRelease, force bool) error {

	urra := nod.Begin(" %s...", ghs.String())
	defer urra.EndWithResult("done")

	for _, release := range releases {

		relDir, err := data.GetAbsReleasesDir(ghs, &release)
		if err != nil {
			return urra.EndWithError(err)
		}

		if _, err := os.Stat(relDir); err == nil && !force {
			urra.EndWithResult("already exists")
			return nil
		}

		if asset := selectAsset(ghs, &release); asset != nil {

			if err := unpackAsset(ghs, &release, asset); err != nil {
				return urra.EndWithError(err)
			}
		}
	}

	return nil
}

func unpackAsset(ghs *data.GitHubSource, release *github_integration.GitHubRelease, asset *github_integration.GitHubAsset) error {

	uaa := nod.Begin(" - unpacking %s, please wait...", asset.Name)
	defer uaa.EndWithResult("done")

	absPackedAssetPath, err := data.GetAbsReleaseAssetPath(ghs, release, asset)
	if err != nil {
		return uaa.EndWithError(err)
	}

	binDir, err := data.GetAbsBinariesDir(ghs, release)
	if err != nil {
		return uaa.EndWithError(err)
	}

	srcFile, err := os.Open(absPackedAssetPath)
	if err != nil {
		return uaa.EndWithError(err)
	}
	defer srcFile.Close()

	if strings.HasSuffix(absPackedAssetPath, tarGzExt) {
		return extractTarGz(srcFile, binDir)
	} else if strings.HasSuffix(absPackedAssetPath, tarXzExt) {
		return extractTarXz(srcFile, binDir)
	} else {
		return uaa.EndWithError(errors.New("archive type is not supported"))
	}
}

func extractTarGz(srcReader io.Reader, dstPath string) error {
	if gzReader, err := gzip.NewReader(srcReader); err == nil {
		return extractTarFiles(gzReader, dstPath)
	} else {
		return err
	}
}

func extractTarXz(srcReader io.Reader, dstPath string) error {
	if xzReader, err := xz.NewReader(srcReader, 0); err == nil {
		return extractTarFiles(xzReader, dstPath)
	} else {
		return err
	}
}

func extractTarFiles(reader io.Reader, dstPath string) error {
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		dstFilePath := filepath.Join(dstPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(dstFilePath); err != nil {
				if err := os.MkdirAll(dstFilePath, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			dstFile, err := os.Create(dstFilePath)
			if err != nil {
				return err
			}

			if _, err := io.Copy(dstFile, tarReader); err != nil {
				dstFile.Close()
				return err
			}

			dstFile.Close()
		}

	}

	return nil
}
