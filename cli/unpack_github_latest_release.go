package cli

import (
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"os"
	"os/exec"
	"path/filepath"
)

func unpackGitHubLatestRelease(operatingSystem vangogh_integration.OperatingSystem, force bool) error {

	ura := nod.NewProgress("unpacking GitHub releases for %s...", operatingSystem)
	defer ura.EndWithResult("done")

	gitHubSources := data.OsGitHubSources(operatingSystem)

	ura.TotalInt(len(gitHubSources))

	for _, ghs := range gitHubSources {

		latestRelease, err := ghs.GetLatestRelease()
		if err != nil {
			return ura.EndWithError(err)
		}

		if latestRelease == nil {
			ura.Increment()
			continue
		}

		binDir, err := data.GetAbsBinariesDir(ghs, latestRelease)
		if err != nil {
			return ura.EndWithError(err)
		}

		if _, err := os.Stat(binDir); err == nil && !force {
			ura.Increment()
			continue
		}

		if asset := ghs.GetAsset(latestRelease); asset != nil {
			if err := unpackAsset(ghs, latestRelease, asset); err != nil {
				return ura.EndWithError(err)
			}
		}

		ura.Increment()
	}

	return nil
}

func unpackAsset(ghs *data.GitHubSource, release *github_integration.GitHubRelease, asset *github_integration.GitHubAsset) error {

	uaa := nod.Begin(" unpacking %s, please wait...", asset.Name)
	defer uaa.EndWithResult("done")

	absPackedAssetPath, err := data.GetAbsReleaseAssetPath(ghs, release, asset)
	if err != nil {
		return uaa.EndWithError(err)
	}

	absBinDir, err := data.GetAbsBinariesDir(ghs, release)
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
	case data.UmuProton.OwnerRepo:
		fallthrough
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
}
