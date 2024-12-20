package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"golang.org/x/exp/slices"
	"net/url"
	"os"
	"path/filepath"
)

func UninstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	operatingSystems, _, _ := OsLangCodeDownloadType(u)
	keepDownloads := q.Has("keep-downloads")
	force := q.Has("force")

	return Uninstall(ids, operatingSystems, keepDownloads, force)
}

func Uninstall(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	keepDownloads bool,
	force bool) error {

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

		if slices.Contains(operatingSystems, vangogh_local_data.MacOS) {
			if err := uninstallMacOsProduct(title, installedAppsDir, bundleName); err != nil {
				return ua.EndWithError(err)
			}
		}

		if slices.Contains(operatingSystems, vangogh_local_data.Windows) {
			if err := uninstallWindowsProduct(title, installedAppsDir, bundleName); err != nil {
				return ua.EndWithError(err)
			}
		}

		if slices.Contains(operatingSystems, vangogh_local_data.Linux) {
			if err := uninstallLinuxProduct(title, installedAppsDir, bundleName); err != nil {
				return ua.EndWithError(err)
			}
		}

		if _, err := kvInstalledMetadata.Cut(id); err != nil {
			return ua.EndWithError(err)
		}

		ua.Increment()
	}

	return nil

}

func uninstallMacOsProduct(title, installationDir, bundleName string) error {

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

func uninstallWindowsProduct(title, installationDir, bundleName string) error {
	return errors.New("uninstalling Windows products is not implemented")
}

func uninstallLinuxProduct(title, installationDir, bundleName string) error {
	return errors.New("uninstalling Linux products is not implemented")
}
