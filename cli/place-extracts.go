package cli

import (
	"github.com/arelate/vangogh_local_data"
	"net/url"
)

func PlaceExtractsHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return PlaceExtracts(ids, operatingSystems, langCodes, downloadTypes, force)
}

func PlaceExtracts(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	return nil

}
