package cli

import (
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/http"
	"path"
	"time"
)

func getGitHubReleases(force bool) error {

	currentOs := data.CurrentOS()

	gra := nod.Begin(" getting GitHub releases for %s...", currentOs)
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

	forceRepoUpdate := force

	for _, repo := range data.OsGitHubSources(currentOs) {

		if ghsu, ok := rdx.GetLastVal(data.GitHubReleasesUpdatedProperty, repo.OwnerRepo); ok && ghsu != "" {
			if ghsut, err := time.Parse(time.RFC3339, ghsu); err == nil {
				if ghsut.AddDate(0, 0, forceGitHubUpdatesDays).Before(time.Now()) {
					forceRepoUpdate = true
				}
			}
		}

		if err := getRepoReleases(repo, kvGitHubReleases, rdx, forceRepoUpdate); err != nil {
			return gra.EndWithError(err)
		}
	}

	return nil
}

func getRepoReleases(ghs *data.GitHubSource, kvGitHubReleases kevlar.KeyValues, rdx kevlar.WriteableRedux, force bool) error {

	grlra := nod.Begin(" %s...", ghs.OwnerRepo)
	defer grlra.EndWithResult("done")

	has, err := kvGitHubReleases.Has(ghs.OwnerRepo)
	if err != nil {
		return grlra.EndWithError(err)
	}

	if has && !force {
		grlra.EndWithResult("skip recently updated")
		return nil
	}

	ghsu := github_integration.ReleasesUrl(path.Split(ghs.OwnerRepo))

	resp, err := http.DefaultClient.Get(ghsu.String())
	if err != nil {
		return grlra.EndWithError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return grlra.EndWithError(errors.New(resp.Status))
	}

	if err := kvGitHubReleases.Set(ghs.OwnerRepo, resp.Body); err != nil {
		return grlra.EndWithError(err)
	}

	ft := time.Now().Format(time.RFC3339)
	return rdx.ReplaceValues(data.GitHubReleasesUpdatedProperty, ghs.OwnerRepo, ft)

}
