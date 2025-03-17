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

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return Uninstall(langCode, rdx, force, ids...)
}

func Uninstall(langCode string, rdx redux.Writeable, force bool, ids ...string) error {

	ua := nod.NewProgress("uninstalling products...")
	defer ua.Done()

	if err := resolveProductTitles(rdx, ids...); err != nil {
		return err
	}

	if !force {
		ua.EndWithResult("this operation requires force flag")
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

	ua.TotalInt(len(ids))

	for _, id := range ids {
		if err = currentOsUninstallProduct(id, langCode, rdx); err != nil {
			return err
		}

		if err = kvOsLangInstalledMetadata.Cut(id); err != nil {
			return err
		}

		ua.Increment()
	}

	if err = unpinInstallParameters(data.CurrentOs(), langCode, rdx, ids...); err != nil {
		return err
	}

	if err = RemoveSteamShortcut(ids...); err != nil {
		return err
	}

	return nil

}

func currentOsUninstallProduct(id, langCode string, rdx redux.Readable) error {
	currentOs := data.CurrentOs()
	switch currentOs {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		if err := nixUninstallProduct(id, langCode, currentOs, rdx); err != nil {
			return err
		}
	case vangogh_integration.Windows:
		if err := windowsUninstallProduct(id, langCode, rdx); err != nil {
			return err
		}
	default:
		panic("unsupported operating system")
	}
	return nil
}
