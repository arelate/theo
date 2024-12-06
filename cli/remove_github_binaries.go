package cli

import (
	"encoding/json"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
)

func RemoveGitHubBinariesHandler(u *url.URL) error {
	q := u.Query()

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	releaseSelector := data.ReleaseSelectorFromUrl(u)
	force := q.Has("force")

	return RemoveGitHubBinaries(operatingSystems, releaseSelector, force)
}

func RemoveGitHubBinaries(operatingSystems []vangogh_local_data.OperatingSystem, releaseSelector *data.GitHubReleaseSelector, force bool) error {

	rba := nod.Begin("removing unpacked GitHub binaries...")
	defer rba.EndWithResult("done")

	PrintReleaseSelector(operatingSystems, releaseSelector)

	gitHubReleasesDir, err := pathways.GetAbsRelDir(data.GitHubReleases)
	if err != nil {
		return rba.EndWithError(err)
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return rba.EndWithError(err)
	}

	githubSources, err := data.AllGitHubSources()
	if err != nil {
		return rba.EndWithError(err)
	}

	for _, os := range operatingSystems {
		for _, repo := range githubSources {

			if repo.OS != os {
				continue
			}

			rcReleases, err := kvGitHubReleases.Get(repo.String())
			if err != nil {
				return rba.EndWithError(err)
			}

			var releases []github_integration.GitHubRelease
			if err := json.NewDecoder(rcReleases).Decode(&releases); err != nil {
				rcReleases.Close()
				return rba.EndWithError(err)
			}

			if err := rcReleases.Close(); err != nil {
				return rba.EndWithError(err)
			}

			selectedReleases := data.SelectReleases(repo, releases, releaseSelector)

			if len(selectedReleases) == 0 {
				continue
			}

			if err := removeRepoBinaries(repo, selectedReleases, force); err != nil {
				return rba.EndWithError(err)
			}
		}
	}

	return nil
}

func removeRepoBinaries(ghs *data.GitHubSource, releases []github_integration.GitHubRelease, force bool) error {

	rrba := nod.Begin(" %s...", ghs.String())
	defer rrba.EndWithResult("done")

	for _, release := range releases {

		rrda := nod.Begin(" - %s...", release.TagName)

		absReleaseBinariesDir, err := data.GetAbsBinariesDir(ghs, &release)
		if err != nil {
			return rrba.EndWithError(err)
		}

		if _, err := os.Stat(absReleaseBinariesDir); os.IsNotExist(err) {
			rrda.EndWithResult("not present")
			continue
		} else if err == nil {
			if !force {
				rrda.EndWithResult("found release binaries, use -force to remove")
				continue
			} else {
				if err := os.RemoveAll(absReleaseBinariesDir); err != nil {
					return rrda.EndWithError(err)
				}
				rrda.EndWithResult("removed")
				continue
			}
		} else {
			return rrda.EndWithError(err)
		}
	}

	return nil
}
