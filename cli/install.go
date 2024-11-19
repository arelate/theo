package cli

import (
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
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

	ia := nod.Begin("installing products...")
	defer ia.EndWithResult("done")

	PrintParams(ids, operatingSystems, langCodes, downloadTypes)

	if err := Backup(); err != nil {
		return err
	}

	if err := Download(ids, operatingSystems, langCodes, downloadTypes, force); err != nil {
		return err
	}

	if err := Validate(ids, operatingSystems, langCodes); err != nil {
		return err
	}

	if err := PinInstalledMetadata(ids, force); err != nil {
		return err
	}

	if err := Extract(ids, operatingSystems, langCodes, downloadTypes, force); err != nil {
		return err
	}

	if err := PlaceExtracts(ids, operatingSystems, langCodes, downloadTypes, force); err != nil {
		return err
	}

	if err := FinalizeInstallation(ids, operatingSystems, langCodes); err != nil {
		return err
	}

	if err := RemoveExtracts(ids, operatingSystems, langCodes, force); err != nil {
		return err
	}

	ia.EndWithResult("done")

	return nil
}
