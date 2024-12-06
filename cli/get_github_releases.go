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
	"time"
)

const (
	forceUpdateDays = 30
)

func GetGitHubReleasesHandler(u *url.URL) error {

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return GetGitHubReleases(operatingSystems, force)
}

func GetGitHubReleases(operatingSystems []vangogh_local_data.OperatingSystem, force bool) error {

	gra := nod.Begin("getting GitHub releases...")
	defer gra.EndWithResult("done")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return gra.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir, data.GitHubReleasesUpdatedProperty)
	if err != nil {
		return gra.EndWithError(err)
	}

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return gra.EndWithError(err)
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return gra.EndWithError(err)
	}

	githubSources, err := data.AllGitHubSources()
	if err != nil {
		return gra.EndWithError(err)
	}

	for _, os := range operatingSystems {

		forceRepoUpdate := force

		for _, repo := range githubSources {

			if repo.OS != os {
				continue
			}

			if ghsu, ok := rdx.GetLastVal(data.GitHubReleasesUpdatedProperty, repo.String()); ok && ghsu != "" {
				if ghsut, err := time.Parse(time.RFC3339, ghsu); err == nil {
					if ghsut.AddDate(0, 0, forceUpdateDays).Before(time.Now()) {
						forceRepoUpdate = true
					}
				}
			}

			if err := getRepoReleases(repo, kvGitHubReleases, rdx, forceRepoUpdate); err != nil {
				return gra.EndWithError(err)
			}
		}
	}

	return nil
}

func getRepoReleases(ghs *data.GitHubSource, kvGitHubReleases kevlar.KeyValues, rdx kevlar.WriteableRedux, force bool) error {

	grlra := nod.Begin(" %s...", ghs.String())
	defer grlra.EndWithResult("done")

	has, err := kvGitHubReleases.Has(ghs.String())
	if err != nil {
		return grlra.EndWithError(err)
	}

	if has && !force {
		grlra.EndWithResult("skip recently updated")
		return nil
	}

	ghsu := github_integration.ReleasesUrl(ghs.Owner, ghs.Repo)

	resp, err := http.DefaultClient.Get(ghsu.String())
	if err != nil {
		return grlra.EndWithError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return grlra.EndWithError(errors.New(resp.Status))
	}

	if err := kvGitHubReleases.Set(ghs.String(), resp.Body); err != nil {
		return grlra.EndWithError(err)
	}

	ft := time.Now().Format(time.RFC3339)
	return rdx.ReplaceValues(data.GitHubReleasesUpdatedProperty, ghs.String(), ft)

}
