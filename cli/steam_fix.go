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

	fixAppId := q.Has("appid")
	revert := q.Has("revert")

	return SteamFix(id, ii, fixAppId, revert)
}

func SteamFix(steamAppId string, ii *InstallInfo, fixAppId, revert bool) error {

	sfa := nod.Begin("applying Steam fixes...")
	defer sfa.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	if fixAppId {
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

	appInfoKv, err := steamGetAppInfoKv(steamAppId, rdx, ii.force)
	if err != nil {
		return err
	}

	defaultSteamEt, err := steamDefaultTask(steamAppId, appInfoKv, ii, rdx)
	if err != nil {
		return err
	}

	exeDir, _ := filepath.Split(defaultSteamEt.exe)

	absSteamAppIdTxtPath := filepath.Join(exeDir, steamAppIdTxt)

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
