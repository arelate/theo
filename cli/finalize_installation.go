package cli

import (
	"github.com/arelate/vangogh_local_data"
	"net/url"
)

func FinalizeInstallationHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, _, _ := OsLangCodeDownloadType(u)

	return FinalizeInstallation(ids, operatingSystems)
}

func FinalizeInstallation(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem) error {

	return nil

}
