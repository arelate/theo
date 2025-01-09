package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
)

func UninstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	force := q.Has("force")

	return Uninstall(ids, force)
}

func Uninstall(ids []string, force bool) error {

	ua := nod.NewProgress("uninstalling products...")
	defer ua.EndWithResult("done")

	if !force {
		ua.EndWithResult("uninstall requires force flag")
		return nil
	}

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return ua.EndWithError(err)
	}

	kvInstalledMetadata, err := kevlar.NewKeyValues(installedMetadataDir, kevlar.JsonExt)
	if err != nil {
		return ua.EndWithError(err)
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return ua.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir,
		data.SetupProperties,
		data.TitleProperty,
		data.BundleNameProperty)
	if err != nil {
		return ua.EndWithError(err)
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return ua.EndWithError(err)
	}

	ua.TotalInt(len(ids))

	for _, id := range ids {

		title, _ := rdx.GetLastVal(data.TitleProperty, id)
		bundleName, _ := rdx.GetLastVal(data.BundleNameProperty, id)

		if err := currentOsUninstallProduct(title, installedAppsDir, bundleName); err != nil {
			return ua.EndWithError(err)
		}

		if _, err := kvInstalledMetadata.Cut(id); err != nil {
			return ua.EndWithError(err)
		}

		ua.Increment()
	}

	if err := RemoveSteamShortcut(ids...); err != nil {
		return ua.EndWithError(err)
	}

	return nil

}

func currentOsUninstallProduct(title, installedAppsDir, bundleName string) error {
	switch data.CurrentOS() {
	case vangogh_local_data.MacOS:
		if err := macOsUninstallProduct(title, installedAppsDir, bundleName); err != nil {
			return err
		}
	case vangogh_local_data.Linux:
		if err := linuxUninstallProduct(title, installedAppsDir, bundleName); err != nil {
			return err
		}
	case vangogh_local_data.Windows:
		if err := windowsUninstallProduct(title, installedAppsDir, bundleName); err != nil {
			return err
		}
	default:
		panic("unsupported operating system")
	}
	return nil
}

func macOsUninstallProduct(title, installationDir, bundleName string) error {

	umpa := nod.Begin(" uninstalling macOS version of %s...", title)
	defer umpa.EndWithResult("done")

	if bundleName == "" {
		return errors.New("product must have bundle name for uninstall")
	}

	bundlePath := filepath.Join(installationDir, vangogh_local_data.MacOS.String(), bundleName)

	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		umpa.EndWithResult("not present")
		return nil
	}

	if err := os.RemoveAll(bundlePath); err != nil {
		return err
	}

	return nil
}

func windowsUninstallProduct(title, installationDir, bundleName string) error {
	return errors.New("uninstalling Windows products is not implemented")
}

func linuxUninstallProduct(title, installationDir, bundleName string) error {
	return errors.New("uninstalling Linux products is not implemented")
}
