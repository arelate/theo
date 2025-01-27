package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
)

const forceGitHubUpdatesDays = 30

func UpdateWineHandler(u *url.URL) error {
	return UpdateWine(u.Query().Has("force"))
}

func UpdateWine(force bool) error {

	currentOs := data.CurrentOs()

	if currentOs == vangogh_integration.Windows {
		err := errors.New("WINE is not required on Windows")
		return err
	}

	uwa := nod.Begin("updating WINE dependencies for %s...", currentOs)
	defer uwa.EndWithResult("done")

	if err := getGitHubReleases(currentOs, force); err != nil {
		return uwa.EndWithError(err)
	}

	if err := cacheGitHubLatestRelease(currentOs, force); err != nil {
		return uwa.EndWithError(err)
	}

	if err := cleanupGitHubReleases(currentOs); err != nil {
		return uwa.EndWithError(err)
	}

	if err := unpackGitHubLatestRelease(currentOs, force); err != nil {
		return uwa.EndWithError(err)
	}

	return nil
}
