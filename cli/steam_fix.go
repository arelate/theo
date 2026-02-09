package cli

import (
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const steamAppIdTxt = "steam_appid.txt"

func SteamFixHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get("id")

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
	}

	placeSteamAppId := q.Has("steam-appid")

	return SteamFix(id, ii, placeSteamAppId)
}

func SteamFix(steamAppId string, ii *InstallInfo, placeSteamAppId bool) error {

	sfa := nod.Begin("applying Steam fixes...")
	defer sfa.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	if placeSteamAppId {

		// https://partner.steamgames.com/doc/sdk/api

		if err = resolveInstallInfo(steamAppId, ii, nil, rdx, installedOperatingSystem); err != nil {
			return err
		}

		var steamAppInstallDir string
		steamAppInstallDir, err = data.AbsSteamAppInstallDir(steamAppId, ii.OperatingSystem, rdx)
		if err != nil {
			return err
		}

		absSteamAppIdTxtPath := filepath.Join(steamAppInstallDir, steamAppIdTxt)
		if _, err = os.Stat(absSteamAppIdTxtPath); err != nil {
			var sait *os.File
			sait, err = os.Create(absSteamAppIdTxtPath)
			if err != nil {
				return err
			}
			defer sait.Close()

			if _, err = io.WriteString(sait, steamAppId); err != nil {
				return err
			}
		}
	}

	return nil
}
