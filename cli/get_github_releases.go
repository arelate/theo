package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/http"
)

func getGitHubReleases(os vangogh_integration.OperatingSystem) error {

	ggra := nod.Begin(" getting GitHub releases for %s...", os)
	defer ggra.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir, data.AllProperties()...)
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

	for _, repo := range vangogh_integration.OperatingSystemGitHubRepos(os) {

		if err = getRepoReleases(repo, kvGitHubReleases, rdx); err != nil {
			return err
		}
	}

	return nil
}

func getRepoReleases(repo string, kvGitHubReleases kevlar.KeyValues, rdx redux.Readable) error {

	grlra := nod.Begin(" %s...", repo)
	defer grlra.Done()

	ghsu, err := data.ServerUrl(rdx, data.ServerGitHubReleasesPath, map[string]string{"repo": repo})
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Get(ghsu.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	return kvGitHubReleases.Set(repo, resp.Body)
}
