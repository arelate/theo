package cli

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"

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

	var steamAppName string
	if san, ok := rdx.GetLastVal(vangogh_integration.TitleProperty, steamAppId); ok && san != "" {
		steamAppName = san
	} else {
		return errors.New("cannot resolve app name for " + steamAppId)
	}

	steamAppsDir := data.Pwd.AbsDirPath(data.SteamApps)

	absSteamInstallDir := filepath.Join(steamAppsDir, ii.OperatingSystem.String(), steamAppName)

	if _, err = os.Stat(absSteamInstallDir); err == nil {
		if err = os.RemoveAll(absSteamInstallDir); err != nil {
			return err
		}
	} else if os.IsNotExist(err) && !ii.force {
		sua.EndWithResult("Steam app directory not found for %s", steamAppId)
		return nil
	} else if err != nil {
		return err
	}

	if err = unpinInstallInfo(steamAppId, ii, rdx); err != nil {
		return err
	}

	if err = removeSteamShortcut(rdx, steamAppId); err != nil {
		return err
	}

	return nil
}
