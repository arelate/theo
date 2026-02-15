package cli

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/author"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func ConnectHandler(u *url.URL) error {

	q := u.Query()

	urlStr := q.Get("url")

	username := q.Get("username")
	password := q.Get("password")

	steam := q.Has("steam")
	reset := q.Has("reset")

	return Connect(urlStr, username, password, steam, reset)
}

func Connect(urlStr, username, password string, steam, reset bool) error {

	ca := nod.Begin("setting up theo connection...")
	defer ca.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	switch steam {
	case true:
		return steamSetupConnection(username, rdx, reset)
	default:
		return vangoghSetupConnection(urlStr, username, password, rdx, reset)
	}
}

func vangoghSetupConnection(urlStr, username, password string, rdx redux.Writeable, reset bool) error {

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	if reset {
		if err := vangoghResetConnection(rdx); err != nil {
			return err
		}
	}

	if err := rdx.ReplaceValues(data.VangoghUrlProperty, data.VangoghUrlProperty, urlStr); err != nil {
		return err
	}

	if err := rdx.ReplaceValues(data.VangoghUsernameProperty, data.VangoghUsernameProperty, username); err != nil {
		return err
	}

	if err := vangoghUpdateSessionToken(password, rdx); err != nil {
		return err
	}

	return vangoghValidateSessionToken(rdx)
}

func vangoghResetConnection(rdx redux.Writeable) error {
	rvca := nod.Begin("resetting vangogh connection...")
	defer rvca.Done()

	for _, vp := range data.VangoghProperties() {
		if err := rdx.CutKeys(vp, vp); err != nil {
			return err
		}
	}

	return nil
}

func vangoghValidateSessionToken(rdx redux.Readable) error {

	tsa := nod.Begin("validating vangogh session token...")
	defer tsa.Done()

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	req, err := data.VangoghRequest(http.MethodPost, data.ApiAuthSessionPath, nil, rdx)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		msg := "session is not valid, please connect again"
		tsa.EndWithResult(msg)
		return errors.New(msg)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	var ste author.SessionTokenExpires

	if err = json.UnmarshalRead(resp.Body, &ste); err != nil {
		return err
	}

	utcNow := time.Now().UTC()

	if utcNow.Before(ste.Expires.Add(-1 * time.Hour * 24)) {
		tsa.EndWithResult("session is valid")
		return nil
	} else {
		msg := "vangogh session expires soon, connect to update"
		tsa.EndWithResult(msg)
		return errors.New(msg)
	}

}

func vangoghUpdateSessionToken(password string, rdx redux.Writeable) error {
	rsa := nod.Begin("updating vangogh session token...")
	defer rsa.Done()

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	var username string
	if up, ok := rdx.GetLastVal(data.VangoghUsernameProperty, data.VangoghUsernameProperty); ok && up != "" {
		username = up
	} else {
		return errors.New("username not found")
	}

	usernamePassword := url.Values{}
	usernamePassword.Set(author.UsernameParam, username)
	usernamePassword.Set(author.PasswordParam, password)

	req, err := data.VangoghRequest(http.MethodPost, data.ApiAuthUserPath, usernamePassword, rdx)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	var ste author.SessionTokenExpires

	if err = json.UnmarshalRead(resp.Body, &ste); err != nil {
		return err
	}

	if err = rdx.ReplaceValues(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty, ste.Token); err != nil {
		return err
	}

	if err = rdx.ReplaceValues(data.VangoghSessionExpiresProperty, data.VangoghSessionExpiresProperty, ste.Expires.Format(http.TimeFormat)); err != nil {
		return err
	}

	return nil
}

func steamSetupConnection(username string, rdx redux.Writeable, reset bool) error {

	if err := rdx.MustHave(data.SteamProperties()...); err != nil {
		return err
	}

	if reset {
		if err := steamResetConnection(rdx); err != nil {
			return err
		}
	}

	switch username {
	case "":
		if un, ok := rdx.GetLastVal(data.SteamUsernameProperty, data.SteamUsernameProperty); ok && un != "" {
			username = un
		} else {
			return errors.New("please provide Steam username")
		}
	default:
		if err := rdx.ReplaceValues(data.SteamUsernameProperty, data.SteamUsernameProperty, username); err != nil {
			return err
		}
	}

	absSteamCmdPath, err := data.AbsSteamCmdBinPath(data.CurrentOs())
	if err != nil {
		return err
	}

	return steamcmd.Login(absSteamCmdPath, username)
}

func steamResetConnection(rdx redux.Writeable) error {
	rsca := nod.Begin("resetting Steam connection...")
	defer rsca.Done()

	if err := rdx.CutKeys(data.SteamUsernameProperty, data.SteamUsernameProperty); err != nil {
		return err
	}

	absSteamCmdPath, err := data.AbsSteamCmdBinPath(data.CurrentOs())
	if err != nil {
		return err
	}

	return steamcmd.Logout(absSteamCmdPath)
}
