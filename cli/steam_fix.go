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

	fixSteamAppId := q.Has("steam-appid")
	revert := q.Has("revert")

	return SteamFix(id, ii, fixSteamAppId, revert)
}

func SteamFix(steamAppId string, ii *InstallInfo, fixSteamAppId, revert bool) error {

	sfa := nod.Begin("applying Steam fixes...")
	defer sfa.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	if fixSteamAppId {
		if err = steamAppIdFix(steamAppId, ii, rdx, revert); err != nil {
			return err
		}
	}

	return nil
}

func steamAppIdFix(steamAppId string, request *InstallInfo, rdx redux.Writeable, revert bool) error {

	saifa := nod.Begin(" applying steam-appid.txt fix...")
	defer saifa.Done()

	// https://partner.steamgames.com/doc/sdk/api

	ii, err := matchInstalledInfo(steamAppId, request, rdx)
	if err != nil {
		return err
	}

	var steamAppInstallDir string
	steamAppInstallDir, err = data.AbsSteamAppInstallDir(steamAppId, ii.OperatingSystem, rdx)
	if err != nil {
		return err
	}

	absSteamAppIdTxtPath := filepath.Join(steamAppInstallDir, steamAppIdTxt)

	switch revert {
	case true:
		if err = os.Remove(absSteamAppIdTxtPath); os.IsNotExist(err) {
			saifa.EndWithResult("not present")
		} else if err != nil {
			return err
		}
	default:
		if _, err = os.Stat(absSteamAppIdTxtPath); os.IsNotExist(err) {
			var sait *os.File
			sait, err = os.Create(absSteamAppIdTxtPath)
			if err != nil {
				return err
			}
			defer sait.Close()

			if _, err = io.WriteString(sait, steamAppId); err != nil {
				return err
			}
		} else if os.IsExist(err) {
			saifa.EndWithResult("already exists")
		}
	}

	return nil
}
