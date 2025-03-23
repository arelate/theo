package data

import (
	_ "embed"
	"github.com/arelate/southern_light/github_integration"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/pathways"
	"path/filepath"
)

const relUmuRunPath = "umu/umu-run"

func UmuRunLatestReleasePath() (string, error) {

	gitHubReleasesDir, err := pathways.GetAbsRelDir(GitHubReleases)
	if err != nil {
		return "", err
	}

	kvGitHubReleases, err := kevlar.New(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return "", err
	}

	latestRelease, err := github_integration.GetLatestRelease(github_integration.UmuLauncherRepo, kvGitHubReleases)
	if err != nil {
		return "", err
	}

	absUmuBinDir, err := GetAbsBinariesDir(github_integration.UmuLauncherRepo, latestRelease)
	if err != nil {
		return "", err
	}

	return filepath.Join(absUmuBinDir, relUmuRunPath), nil
}

func UmuProtonLatestReleasePath() (string, error) {

	gitHubReleasesDir, err := pathways.GetAbsRelDir(GitHubReleases)
	if err != nil {
		return "", err
	}

	kvGitHubReleases, err := kevlar.New(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return "", err
	}

	latestRelease, err := github_integration.GetLatestRelease(github_integration.UmuProtonRepo, kvGitHubReleases)
	if err != nil {
		return "", err
	}

	umuProtonDir, err := GetAbsBinariesDir(github_integration.UmuProtonRepo, latestRelease)
	if err != nil {
		return "", err
	}

	// won't sanitize TagName here as it's coming from unpacked release (as provided by the repo owner)
	return filepath.Join(umuProtonDir, latestRelease.TagName), nil
}
