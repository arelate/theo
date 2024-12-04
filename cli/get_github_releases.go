package cli

import (
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/http"
	"net/url"
)

func GetGitHubReleasesHandler(u *url.URL) error {

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return GetGitHubReleases(operatingSystems, force)
}

func GetGitHubReleases(operatingSystems []vangogh_local_data.OperatingSystem, force bool) error {

	glra := nod.Begin("getting latest GitHub releases...")
	defer glra.EndWithResult("done")

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return glra.EndWithError(err)
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return glra.EndWithError(err)
	}

	for _, os := range operatingSystems {
		for _, repo := range data.OperatingSystemRepos[os] {
			has, err := kvGitHubReleases.Has(repo.String())
			if err != nil {
				return glra.EndWithError(err)
			}

			if has && !force {
				continue
			}

			if err := getRepoLatestReleases(&repo, kvGitHubReleases); err != nil {
				return glra.EndWithError(err)
			}
		}
	}

	return nil
}

func getRepoLatestReleases(ghr *data.GitHubRepository, kvGitHubReleases kevlar.KeyValues) error {

	grlra := nod.Begin(" %s...", ghr.String())
	grlra.EndWithResult("done")

	ghru := github_integration.ReleasesUrl(ghr.Owner, ghr.Repo)

	resp, err := http.DefaultClient.Get(ghru.String())
	if err != nil {
		return grlra.EndWithError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return grlra.EndWithError(errors.New(resp.Status))
	}

	return kvGitHubReleases.Set(ghr.String(), resp.Body)
}
