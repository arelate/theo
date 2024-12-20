package cli

import (
	"errors"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"golang.org/x/exp/slices"
	"net/url"
)

func UpdateWineHandler(u *url.URL) error {

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return UpdateWine(operatingSystems, force)
}

func UpdateWine(operatingSystems []vangogh_local_data.OperatingSystem, force bool) error {

	uwa := nod.Begin("updating WINE...")
	defer uwa.EndWithResult("done")

	if slices.Contains(operatingSystems, vangogh_local_data.Windows) {
		err := errors.New("WINE is not required on Windows")
		return uwa.EndWithError(err)
	}

	if err := CheckGstreamer(); err != nil {
		return uwa.EndWithError(err)
	}

	if err := GetGitHubReleases(operatingSystems, force); err != nil {
		return uwa.EndWithError(err)
	}

	if err := CacheGitHubReleases(operatingSystems, nil, force); err != nil {
		return uwa.EndWithError(err)
	}

	if err := CleanupGitHubReleases(operatingSystems); err != nil {
		return uwa.EndWithError(err)
	}

	if err := UnpackGitHubReleases(operatingSystems, nil, force); err != nil {
		return uwa.EndWithError(err)
	}

	return nil
}
