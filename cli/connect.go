package cli

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

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

	reset := q.Has("reset")

	return VangoghConnect(urlStr, username, password, reset)
}

func VangoghConnect(
	urlStr string,
	username, password string,
	reset bool) error {

	vca := nod.Begin("connecting to vangogh...")
	defer vca.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.VangoghProperties()...)
	if err != nil {
		return err
	}

	if reset {
		if err = resetVangoghConnection(rdx); err != nil {
			return err
		}
	}

	if err = rdx.ReplaceValues(data.VangoghUrlProperty, data.VangoghUrlProperty, urlStr); err != nil {
		return err
	}

	if err = rdx.ReplaceValues(data.VangoghUsernameProperty, data.VangoghUsernameProperty, username); err != nil {
		return err
	}

	if err = vangoghUpdateSessionToken(password, rdx); err != nil {
		return err
	}

	if err = vangoghValidateSessionToken(rdx); err != nil {
		return err
	}

	return nil
}

func resetVangoghConnection(rdx redux.Writeable) error {
	rvca := nod.Begin("resetting vangogh connection...")
	defer rvca.Done()

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

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

	if err = json.NewDecoder(resp.Body).Decode(&ste); err != nil {
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

	if err = json.NewDecoder(resp.Body).Decode(&ste); err != nil {
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
