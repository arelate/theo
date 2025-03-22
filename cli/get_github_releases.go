package cli

import (
	"errors"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/http"
	"strings"
	"time"
)

const forceGitHubUpdatesDays = 30

func getGitHubReleases(os vangogh_integration.OperatingSystem, force bool) error {

	ggra := nod.Begin(" getting GitHub releases for %s...", os)
	defer ggra.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, vangogh_integration.GitHubReleasesUpdatedProperty)
	if err != nil {
		return err
	}

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return err
	}

	kvGitHubReleases, err := kevlar.New(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	forceRepoUpdate := force

	for _, repo := range vangogh_integration.OperatingSystemGitHubSources(os) {

		if ghsu, ok := rdx.GetLastVal(data.GitHubReleasesUpdatedProperty, repo.OwnerRepo); ok && ghsu != "" {
			if ghsut, err := time.Parse(time.RFC3339, ghsu); err == nil {
				if ghsut.AddDate(0, 0, forceGitHubUpdatesDays).Before(time.Now()) {
					forceRepoUpdate = true
				}
			}
		}

		if err = getRepoReleases(repo, kvGitHubReleases, rdx, forceRepoUpdate); err != nil {
			return err
		}
	}

	return nil
}

func getRepoReleases(ghs *github_integration.GitHubSource, kvGitHubReleases kevlar.KeyValues, rdx redux.Writeable, force bool) error {

	grlra := nod.Begin(" %s...", ghs.OwnerRepo)
	defer grlra.Done()

	if kvGitHubReleases.Has(ghs.OwnerRepo) && !force {
		grlra.EndWithResult("skip recently updated")
		return nil
	}

	owner, repo, _ := strings.Cut(ghs.OwnerRepo, "/")
	ghsu := github_integration.ReleasesUrl(owner, repo)

	resp, err := http.DefaultClient.Get(ghsu.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	if err = kvGitHubReleases.Set(ghs.OwnerRepo, resp.Body); err != nil {
		return err
	}

	ft := time.Now().Format(time.RFC3339)
	return rdx.ReplaceValues(data.GitHubReleasesUpdatedProperty, ghs.OwnerRepo, ft)
}
