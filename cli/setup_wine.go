package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
	"time"
)

func SetupWineHandler(u *url.URL) error {
	return SetupWine(u.Query().Has("force"))
}

func SetupWine(force bool) error {

	start := time.Now().UTC().Unix()

	currentOs := data.CurrentOs()

	if currentOs == vangogh_integration.Windows {
		err := errors.New("WINE is not required on Windows")
		return err
	}

	uwa := nod.Begin("setting up WINE for %s...", currentOs)
	defer uwa.Done()

	if err := getGitHubReleases(currentOs); err != nil {
		return err
	}

	if err := downloadGitHubLatestRelease(currentOs, start, force); err != nil {
		return err
	}

	if err := cleanupGitHubReleases(currentOs, start, force); err != nil {
		return err
	}

	if err := unpackGitHubLatestRelease(currentOs, start, force); err != nil {
		return err
	}

	return nil
}
