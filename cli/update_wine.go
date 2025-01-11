package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"net/url"
	"os"
)

const gstreamerFrameworkPath = "/Library/Frameworks/GStreamer.framework"

const forceGitHubUpdatesDays = 30

func UpdateWineHandler(u *url.URL) error {
	return UpdateWine(u.Query().Has("force"))
}

func UpdateWine(force bool) error {

	currentOs := data.CurrentOS()

	if currentOs == vangogh_local_data.Windows {
		err := errors.New("WINE is not required on Windows")
		return err
	}

	uwa := nod.Begin("updating WINE for %s...", currentOs)
	defer uwa.EndWithResult("done")

	if err := checkGstreamer(); err != nil {
		return uwa.EndWithError(err)
	}

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

func checkGstreamer() error {

	if data.CurrentOS() != vangogh_local_data.MacOS {
		return nil
	}

	cga := nod.Begin(" checking whether GStreamer.framework is installed...")
	defer cga.EndWithResult("done")

	if _, err := os.Stat(gstreamerFrameworkPath); err == nil {
		cga.EndWithResult("found")
		return nil
	} else if os.IsNotExist(err) {
		cga.EndWithResult("not found. Download it at https://gstreamer.freedesktop.org/download")
		return nil
	} else {
		return cga.EndWithError(err)
	}
}
