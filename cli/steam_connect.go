package cli

import (
	"net/url"

	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func SteamConnectHandler(u *url.URL) error {

	q := u.Query()

	username := q.Get("username")
	reset := q.Has("reset")

	return SteamConnect(username, reset)
}

func SteamConnect(username string, reset bool) error {
	sca := nod.Begin("connecting to Steam...")
	defer sca.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.SteamProperties()...)
	if err != nil {
		return err
	}

	if reset {
		if err = resetSteamConnection(rdx); err != nil {
			return err
		}
	}

	if err = rdx.ReplaceValues(data.SteamUsernameProperty, data.SteamUsernameProperty, username); err != nil {
		return err
	}

	return steamCmdLogin(username)
}

func resetSteamConnection(rdx redux.Writeable) error {
	rsca := nod.Begin("resetting Steam connection...")
	defer rsca.Done()

	if err := rdx.MustHave(data.SteamProperties()...); err != nil {
		return err
	}

	if err := rdx.CutKeys(data.SteamUsernameProperty, data.SteamUsernameProperty); err != nil {
		return err
	}

	return steamCmdLogout()
}
