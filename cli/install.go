package cli

import (
	"errors"
	"github.com/arelate/vangogh_local_data"
	"net/url"
)

func InstallHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return Install(ids, operatingSystems, langCodes, downloadTypes, force)
}

func Install(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	PrintParams(ids, operatingSystems, langCodes, downloadTypes)

	return errors.New("install cmd is not implemented")
}
