package cli

import (
	"net/url"

	"github.com/boggydigital/nod"
)

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

	}

	return nil
}
