package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func UninstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)

	_, langCodes, _ := OsLangCodeDownloadType(u)

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	force := q.Has("force")

	return Uninstall(langCode, force, ids...)
}

func Uninstall(langCode string, force bool, ids ...string) error {

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

		if err := currentOsUninstallProduct(title, installedAppsDir, langCode, bundleName); err != nil {
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

func currentOsUninstallProduct(title, installedAppsDir, langCode, bundleName string) error {
	currentOs := data.CurrentOS()
	switch currentOs {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		if err := nixUninstallProduct(title, currentOs, installedAppsDir, langCode, bundleName); err != nil {
			return err
		}
	case vangogh_integration.Windows:
		if err := windowsUninstallProduct(title, installedAppsDir, langCode, bundleName); err != nil {
			return err
		}
	default:
		panic("unsupported operating system")
	}
	return nil
}
