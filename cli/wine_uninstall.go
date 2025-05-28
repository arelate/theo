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

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return WineUninstall(langCode, rdx, archive, force, ids...)
}

func WineUninstall(langCode string, rdx redux.Writeable, archive, force bool, ids ...string) error {

	wua := nod.NewProgress("uninstalling WINE installed products...")
	defer wua.Done()

	if !force {
		wua.EndWithResult("this operation requires force flag")
		return nil
	}

	installedDetailsDir, err := pathways.GetAbsRelDir(data.InstalledDetails)
	if err != nil {
		return err
	}

	osLangInstalledDetailsDir := filepath.Join(installedDetailsDir, data.OsLangCode(vangogh_integration.Windows, langCode))

	kvOsLangInstalledDetails, err := kevlar.New(osLangInstalledDetailsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	var flattened bool
	if ids, flattened, err = gameProductTypesFlatMap(rdx, force, ids...); err != nil {
		return err
	} else if flattened {
		wua.EndWithResult("uninstalling PACK included games")
		return Uninstall(langCode, rdx, force, ids...)
	}

	if err = RemovePrefix(langCode, archive, force, ids...); err != nil {
		return err
	}

	if err = DeletePrefixEnv(langCode, force, ids...); err != nil {
		return err
	}

	for _, id := range ids {
		if err = kvOsLangInstalledDetails.Cut(id); err != nil {
			return err
		}

		wua.Increment()
	}

	if err = unpinInstallParameters(vangogh_integration.Windows, langCode, rdx, ids...); err != nil {
		return err
	}

	if err := RemoveSteamShortcut(rdx, ids...); err != nil {
		return err
	}

	return nil

}
