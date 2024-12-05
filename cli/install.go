package cli

import (
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"net/url"
)

func InstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	native := q.Has("native")
	keepDownloads := q.Has("keep-downloads")
	sign := q.Has("sign")
	force := q.Has("force")

	return Install(ids, operatingSystems, langCodes, downloadTypes, native, keepDownloads, sign, force)
}

func Install(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	native bool,
	keepDownloads bool,
	sign bool,
	force bool) error {

	ia := nod.Begin("installing products...")
	defer ia.EndWithResult("done")

	vangogh_local_data.PrintParams(ids, operatingSystems, langCodes, downloadTypes, true)

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

	if native {

		if err := NativeInstall(ids, operatingSystems, langCodes, downloadTypes, force); err != nil {
			return err
		}

	} else {

		if err := Extract(ids, operatingSystems, langCodes, downloadTypes, force); err != nil {
			return err
		}

		if err := PlaceExtracts(ids, operatingSystems, langCodes, downloadTypes, force); err != nil {
			return err
		}

		if err := FinalizeInstallation(ids, operatingSystems, langCodes, sign); err != nil {
			return err
		}

		if err := RemoveExtracts(ids, operatingSystems, langCodes, force); err != nil {
			return err
		}

	}

	if !keepDownloads {
		if err := RemoveDownloads(ids, operatingSystems, langCodes, force); err != nil {
			return err
		}
	}

	if err := RevealInstalled(ids, operatingSystems, langCodes); err != nil {
		return err
	}

	return nil
}
