package cli

import (
	"encoding/json"
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"os"
	"os/exec"
	"path/filepath"
)

func unpackGitHubLatestRelease(os vangogh_integration.OperatingSystem, force bool) error {

	ura := nod.Begin("unpacking GitHub releases for %s...", os)
	defer ura.EndWithResult("done")

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return ura.EndWithError(err)
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return ura.EndWithError(err)
	}

	for _, repo := range data.OsGitHubSources(os) {

		rcReleases, err := kvGitHubReleases.Get(repo.OwnerRepo)
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

		var latestRelease *github_integration.GitHubRelease
		if len(releases) > 0 {
			latestRelease = &releases[0]
		}

		if err := unpackRepoLatestRelease(repo, latestRelease, force); err != nil {
			return ura.EndWithError(err)
		}
	}

	return nil
}

func unpackRepoLatestRelease(ghs *data.GitHubSource, release *github_integration.GitHubRelease, force bool) error {

	urra := nod.Begin(" %s...", ghs.OwnerRepo)
	defer urra.EndWithResult("done")

	binDir, err := data.GetAbsBinariesDir(ghs)
	if err != nil {
		return urra.EndWithError(err)
	}

	if _, err := os.Stat(binDir); err == nil && !force {
		urra.EndWithResult("already exists")
		return nil
	}

	if asset := ghs.GetAsset(release); asset != nil {

		if err := unpackAsset(ghs, release, asset); err != nil {
			return urra.EndWithError(err)
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

	absBinDir, err := data.GetAbsBinariesDir(ghs)
	if err != nil {
		return uaa.EndWithError(err)
	}

	return unpackGitHubSource(ghs, absPackedAssetPath, absBinDir)
}

func untar(srcPath, dstPath string) error {

	if _, err := os.Stat(dstPath); err != nil {
		if err := os.MkdirAll(dstPath, 0755); err != nil {
			return err
		}
	}

	cmd := exec.Command("tar", "-xf", srcPath, "-C", dstPath)
	return cmd.Run()
}

func unzip(srcPath, dstPath string) error {
	if _, err := os.Stat(dstPath); err != nil {
		if err := os.MkdirAll(dstPath, 0755); err != nil {
			return err
		}
	}

	cmd := exec.Command("unzip", srcPath, "-d", dstPath)
	return cmd.Run()
}

func unpackGitHubSource(ghs *data.GitHubSource, absSrcAssetPath, absDstPath string) error {
	switch ghs.OwnerRepo {
	case data.GeProtonCustom.OwnerRepo:
		return untar(absSrcAssetPath, absDstPath)
	case data.UmuLauncher.OwnerRepo:
		// first - unzip Zipapp.zip
		if err := unzip(absSrcAssetPath, absDstPath); err != nil {
			return err
		}
		// second - untar Zipapp.tar in the binaries dir
		absSrcAssetPath = filepath.Join(absDstPath, "Zipapp.tar")
		return untar(absSrcAssetPath, absDstPath)
	default:
		return errors.New("unknown GitHub source: " + ghs.OwnerRepo)
	}
	return nil
}
