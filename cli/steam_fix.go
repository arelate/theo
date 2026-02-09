package cli

import (
	"net/url"

	"github.com/boggydigital/nod"
)

const steamAppIdTxt = "steam_appid.txt"

func SteamFixHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get("id")
	addSteamAppId := q.Has("steam-appid")

	return SteamFix(id, addSteamAppId)
}

func SteamFix(steamAppId string, addSteamAppId bool) error {

	sfa := nod.Begin("applying Steam fixes...")
	defer sfa.Done()

	if addSteamAppId {

		//// https://partner.steamgames.com/doc/sdk/api
		//absSteamAppIdTxtPath := filepath.Join(absInstallDir, steamAppIdTxt)
		//if _, err = os.Stat(absSteamAppIdTxtPath); err != nil {
		//	var sait *os.File
		//	sait, err = os.Create(absSteamAppIdTxtPath)
		//	if err != nil {
		//		return err
		//	}
		//	defer sait.Close()
		//
		//	if _, err = io.WriteString(sait, steamAppId); err != nil {
		//		return err
		//	}
		//}

	}

	return nil
}
