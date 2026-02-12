package cli

import (
	"net/url"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func SteamUninstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        defaultLangCode,
		SteamInstall:    true,
		force:           q.Has("force"),
	}

	return SteamUninstall(id, ii)
}

func SteamUninstall(steamAppId string, ii *InstallInfo) error {

	sua := nod.Begin("unisntalling Steam appId %s...", steamAppId)
	defer sua.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	if !ii.force {
		sua.EndWithResult("steam-uninstall requires -force parameter")
		return nil
	}

	if err = resolveInstallInfo(steamAppId, ii, nil, rdx, installedOperatingSystem); err != nil {
		return err
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, steamAppId); ok {

		var installInfo *InstallInfo
		installInfo, _, err = matchInstallInfo(ii, installedInfoLines...)
		if err != nil {
			return err
		}

		if installInfo == nil {
			sua.EndWithResult("no install info found for %s %s", steamAppId, ii.OperatingSystem)
			return nil
		}

	}

	steamAppInstallDir, err := data.AbsSteamAppInstallDir(steamAppId, ii.OperatingSystem, rdx)
	if err != nil {
		return err
	}

	var username string
	if un, ok := rdx.GetLastVal(data.SteamUsernameProperty, data.SteamUsernameProperty); ok && un != "" {
		username = un
	}

	if err = steamCmdAppUninstall(steamAppId, ii.OperatingSystem, steamAppInstallDir, username); err != nil {
		return err
	}

	//if err = unpinInstallInfo(steamAppId, ii, rdx); err != nil {
	//	return err
	//}

	if err = removeSteamShortcut(rdx, steamAppId); err != nil {
		return err
	}

	return nil
}
