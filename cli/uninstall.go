package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path/filepath"
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
	defer ua.Done()

	if !force {
		ua.EndWithResult("this operation requires force flagujjjjjjji")
		return nil
	}

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return err
	}

	osLangInstalledMetadataDir := filepath.Join(installedMetadataDir, data.OsLangCode(data.CurrentOs(), langCode))

	kvOsLangInstalledMetadata, err := kevlar.New(osLangInstalledMetadataDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir,
		data.ServerConnectionProperties,
		data.TitleProperty,
		data.BundleNameProperty)
	if err != nil {
		return err
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	ua.TotalInt(len(ids))

	for _, id := range ids {

		title, _ := rdx.GetLastVal(data.TitleProperty, id)
		bundleName, _ := rdx.GetLastVal(data.BundleNameProperty, id)

		if err := currentOsUninstallProduct(title, installedAppsDir, langCode, bundleName); err != nil {
			return err
		}

		if err = kvOsLangInstalledMetadata.Cut(id); err != nil {
			return err
		}

		ua.Increment()
	}

	if err = unpinInstallParameters(data.CurrentOs(), langCode, ids...); err != nil {
		return err
	}

	if err = RemoveSteamShortcut(ids...); err != nil {
		return err
	}

	return nil

}

func currentOsUninstallProduct(title, installedAppsDir, langCode, bundleName string) error {
	currentOs := data.CurrentOs()
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
