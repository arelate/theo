package cli

import (
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
	archive := !q.Has("no-archive")
	force := q.Has("force")

	return WineUninstall(langCode, archive, force, ids...)
}

func WineUninstall(langCode string, archive, force bool, ids ...string) error {

	wua := nod.NewProgress("uninstalling WINE installed products...")
	defer wua.EndWithResult("done")

	if !force {
		wua.EndWithResult("uninstall requires force flag")
		return nil
	}

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return wua.EndWithError(err)
	}

	osInstalledMetadataDir := filepath.Join(installedMetadataDir, data.CurrentOS().String())

	kvOsInstalledMetadata, err := kevlar.NewKeyValues(osInstalledMetadataDir, kevlar.JsonExt)
	if err != nil {
		return wua.EndWithError(err)
	}

	if err := RemovePrefix(langCode, archive, force, ids...); err != nil {
		return wua.EndWithError(err)
	}

	for _, id := range ids {
		if _, err := kvOsInstalledMetadata.Cut(id); err != nil {
			return wua.EndWithError(err)
		}

		wua.Increment()
	}

	if err := RemoveSteamShortcut(ids...); err != nil {
		return wua.EndWithError(err)
	}

	return nil

}
