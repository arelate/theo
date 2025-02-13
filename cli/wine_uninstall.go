package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

func WineUninstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)

	_, langCodes, _ := OsLangCodeDownloadType(u)

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}
	archive := q.Has("archive")
	force := q.Has("force")

	return WineUninstall(langCode, archive, force, ids...)
}

func WineUninstall(langCode string, archive, force bool, ids ...string) error {

	wua := nod.NewProgress("uninstalling WINE installed products...")
	defer wua.Done()

	if err := resolveProductTitles(ids...); err != nil {
		return err
	}

	if !force {
		wua.EndWithResult("this operation requires force flag")
		return nil
	}

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return err
	}

	osLangInstalledMetadataDir := filepath.Join(installedMetadataDir, data.OsLangCode(vangogh_integration.Windows, langCode))

	kvOsLangInstalledMetadata, err := kevlar.New(osLangInstalledMetadataDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if err := RemovePrefix(langCode, archive, force, ids...); err != nil {
		return err
	}

	if err := DeletePrefixEnv(langCode, force, ids...); err != nil {
		return err
	}

	for _, id := range ids {
		if err = kvOsLangInstalledMetadata.Cut(id); err != nil {
			return err
		}

		wua.Increment()
	}

	if err = unpinInstallParameters(vangogh_integration.Windows, langCode, ids...); err != nil {
		return err
	}

	if err := RemoveSteamShortcut(ids...); err != nil {
		return err
	}

	return nil

}
