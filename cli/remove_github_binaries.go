package cli

import (
	"github.com/arelate/vangogh_local_data"
	"net/url"
)

func RemoveGitHubBinariesHandler(u *url.URL) error {
	q := u.Query()

	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	releaseSelector := releaseSelectorFromUrl(u)
	force := q.Has("force")

	return RemoveGitHubBinaries(operatingSystems, releaseSelector, force)
}

func RemoveGitHubBinaries(operatingSystems []vangogh_local_data.OperatingSystem, releaseSelector *GitHubReleaseSelector, force bool) error {
	return nil
}
