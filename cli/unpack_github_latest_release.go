package cli

import (
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"os"
	"os/exec"
)

func unpackGitHubLatestRelease(operatingSystem vangogh_integration.OperatingSystem, force bool) error {

	ura := nod.NewProgress("unpacking GitHub releases for %s...", operatingSystem)
	defer ura.Done()

	gitHubRepos := vangogh_integration.OperatingSystemGitHubRepos(operatingSystem)

	ura.TotalInt(len(gitHubRepos))

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return err
	}

	kvGitHubReleases, err := kevlar.New(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	for _, repo := range gitHubRepos {

		latestRelease, err := github_integration.GetLatestRelease(repo, kvGitHubReleases)
		if err != nil {
			return err
		}

		if latestRelease == nil {
			ura.Increment()
			continue
		}

		binDir, err := data.GetAbsBinariesDir(repo, latestRelease)
		if err != nil {
			return err
		}

		if _, err := os.Stat(binDir); err == nil && !force {
			ura.Increment()
			continue
		}

		if asset := github_integration.GetReleaseAsset(repo, latestRelease); asset != nil {
			if err := unpackAsset(repo, latestRelease, asset); err != nil {
				return err
			}
		}

		ura.Increment()
	}

	return nil
}

func unpackAsset(repo string, release *github_integration.GitHubRelease, asset *github_integration.GitHubAsset) error {

	uaa := nod.Begin(" unpacking %s, please wait...", asset.Name)
	defer uaa.Done()

	absPackedAssetPath, err := data.GetAbsReleaseAssetPath(repo, release, asset)
	if err != nil {
		return err
	}

	absBinDir, err := data.GetAbsBinariesDir(repo, release)
	if err != nil {
		return err
	}

	return unpackGitHubSource(repo, absPackedAssetPath, absBinDir)
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

func unpackGitHubSource(repo string, absSrcAssetPath, absDstPath string) error {
	switch repo {
	case github_integration.UmuProtonRepo:
		fallthrough
	case github_integration.UmuLauncherRepo:
		return untar(absSrcAssetPath, absDstPath)
	default:
		return errors.New("unknown GitHub source: " + repo)
	}
}
