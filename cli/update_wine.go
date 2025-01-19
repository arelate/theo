package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
)

const gstreamerFrameworkPath = "/Library/Frameworks/GStreamer.framework"

const forceGitHubUpdatesDays = 30

func UpdateWineHandler(u *url.URL) error {
	return UpdateWine(u.Query().Has("force"))
}

func UpdateWine(force bool) error {

	currentOs := data.CurrentOS()

	if currentOs == vangogh_integration.Windows {
		err := errors.New("WINE is not required on Windows")
		return err
	}

	uwa := nod.Begin("updating WINE dependencies for %s...", currentOs)
	defer uwa.EndWithResult("done")

	if err := getGitHubReleases(force); err != nil {
		return uwa.EndWithError(err)
	}

	if err := cacheGitHubLatestRelease(force); err != nil {
		return uwa.EndWithError(err)
	}

	if err := cleanupGitHubReleases(); err != nil {
		return uwa.EndWithError(err)
	}

	if err := unpackGitHubLatestRelease(force); err != nil {
		return uwa.EndWithError(err)
	}

	return nil
}
