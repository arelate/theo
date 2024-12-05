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
)

func UnpackGitHubReleasesHandler(u *url.URL) error {

	q := u.Query()

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	releaseSelector := releaseSelectorFromUrl(u)
	force := q.Has("force")

	return UnpackGitHubReleases(operatingSystems, releaseSelector, force)
}

func UnpackGitHubReleases(operatingSystems []vangogh_local_data.OperatingSystem, releaseSelector *GitHubReleaseSelector, force bool) error {
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

			selectedReleases := selectReleases(&repo, releases, releaseSelector)

			if len(selectedReleases) == 0 {
				err = errors.New("")
				return ura.EndWithError(err)
			}

			//if err := cacheRepoReleases(&repo, selectedReleases, dc, force); err != nil {
			//	return ura.EndWithError(err)
			//}
		}
	}

	return nil
}
