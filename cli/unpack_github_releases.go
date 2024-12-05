package cli

import (
	"encoding/json"
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

const (
	tarXzExt = ".tar.xz"
	tarGzExt = ".tar.gz"
)

func UnpackGitHubReleasesHandler(u *url.URL) error {

	q := u.Query()

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	releaseSelector := data.ReleaseSelectorFromUrl(u)
	force := q.Has("force")

	return UnpackGitHubReleases(operatingSystems, releaseSelector, force)
}

func UnpackGitHubReleases(operatingSystems []vangogh_local_data.OperatingSystem, releaseSelector *data.GitHubReleaseSelector, force bool) error {

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

			selectedReleases := data.SelectReleases(&repo, releases, releaseSelector)

			if len(selectedReleases) == 0 {
				continue
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

		binDir, err := data.GetAbsBinariesDir(ghs, &release)
		if err != nil {
			return urra.EndWithError(err)
		}

		if _, err := os.Stat(binDir); err == nil && !force {
			urra.EndWithResult("already exists")
			return nil
		}

		if asset := data.SelectAsset(ghs, &release); asset != nil {

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

	if strings.HasSuffix(absPackedAssetPath, tarGzExt) {
		return extractTar(absPackedAssetPath, binDir)
	} else if strings.HasSuffix(absPackedAssetPath, tarXzExt) {
		return extractTar(absPackedAssetPath, binDir)
	} else {
		return uaa.EndWithError(errors.New("archive type is not supported"))
	}
}

func extractTar(srcPath, dstPath string) error {

	if _, err := os.Stat(dstPath); err != nil {
		if err := os.MkdirAll(dstPath, 0755); err != nil {
			return err
		}
	}

	cmd := exec.Command("tar", "-xf", srcPath, "-C", dstPath)
	return cmd.Run()
}
