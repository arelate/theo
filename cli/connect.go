package cli

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/author"
	"github.com/boggydigital/coost"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	egsCookiesFilename = "egs-cookies.json"
	getVerifyTokenKey  = "get-verify-token"
)

func ConnectHandler(u *url.URL) error {

	q := u.Query()

	urlStr := q.Get("url")

	username := q.Get("username")
	password := q.Get("password")

	var origin data.Origin

	if q.Has("steam") {
		origin = data.SteamOrigin
	} else if q.Has("epic-games") {
		origin = data.EpicGamesOrigin
	} else {
		origin = data.VangoghOrigin
	}

	cookie := q.Get("cookie")

	reset := q.Has("reset")

	return Connect(urlStr, username, password, cookie, origin, reset)
}

func Connect(urlStr, username, password, cookie string, origin data.Origin, reset bool) error {

	ca := nod.Begin("setting up theo connection...")
	defer ca.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	switch origin {
	case data.VangoghOrigin:
		return vangoghSetupConnection(urlStr, username, password, rdx, reset)
	case data.SteamOrigin:
		if password != "" {
			return errors.New("steam password will be requested by SteamCMD")
		}
		return steamSetupConnection(username, rdx, reset)
	case data.EpicGamesOrigin:
		return epicGamesSetupConnection(cookie, reset)
	default:
		return origin.ErrUnsupportedOrigin()
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

	ssca := nod.Begin("connecting to Steam...")
	defer ssca.Done()

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

func epicGamesGetClient() (*http.Client, error) {

	egsCookieDir := data.Pwd.AbsRelDirPath(data.EgsCookie, data.Metadata)
	egsCookiePath := filepath.Join(egsCookieDir, egsCookiesFilename)

	jar, err := coost.Read(egs_integration.HostUrl(), egsCookiePath)
	if err != nil {
		return nil, err
	}

	client := http.DefaultClient
	client.Jar = jar

	return client, nil
}

func epicGamesSetupConnection(cookieStr string, reset bool) error {

	egsca := nod.Begin("connecting to EGS...")
	defer egsca.Done()

	var err error

	if reset {
		if err = epicGamesResetConnection(); err != nil {
			return err
		}
	}

	var accessToken string

	if cookieStr != "" {
		if accessToken, err = epicGamesGetAccessToken(cookieStr); err != nil {
			return err
		}
	}

	return epicGamesVerifyToken(accessToken)
}

func epicGamesGetAccessToken(cookieStr string) (string, error) {

	eggata := nod.Begin("getting EGS access token...")
	defer eggata.Done()

	egsCookieDir := data.Pwd.AbsRelDirPath(data.EgsCookie, data.Metadata)
	egsCookiePath := filepath.Join(egsCookieDir, egsCookiesFilename)

	egsVerifyTokenDir := data.Pwd.AbsRelDirPath(data.EgsVerifyToken, data.Metadata)
	kvVerifyToken, err := kevlar.New(egsVerifyTokenDir, kevlar.JsonExt)
	if err != nil {
		return "", err
	}

	if err = coost.Import(cookieStr, egs_integration.HostUrl(), egsCookiePath); err != nil {
		return "", err
	}

	if kvVerifyToken.Has(getVerifyTokenKey) {
		if err = kvVerifyToken.Cut(getVerifyTokenKey); err != nil {
			return "", err
		}
	}

	var client *http.Client
	client, err = epicGamesGetClient()
	if err != nil {
		return "", err
	}

	var arr *egs_integration.GetApiRedirectResponse
	arr, err = egs_integration.GetApiRedirect(client)
	if err != nil {
		return "", err
	}

	var ptr *egs_integration.PostTokenResponse
	ptr, err = egs_integration.PostToken(arr.AuthorizationCode, client)
	if err != nil {
		return "", err
	}

	if ptr.AccessToken == "" {
		return "", errors.New("failed to get epic games access token")
	}

	return ptr.AccessToken, nil
}

func epicGamesGetStoredVerifyToken() (string, error) {

	eggsvta := nod.Begin("getting stored EGS access token...")
	defer eggsvta.Done()

	egsVerifyTokenDir := data.Pwd.AbsRelDirPath(data.EgsVerifyToken, data.Metadata)
	kvVerifyToken, err := kevlar.New(egsVerifyTokenDir, kevlar.JsonExt)
	if err != nil {
		return "", err
	}

	var rcVerifyToken io.ReadCloser
	rcVerifyToken, err = kvVerifyToken.Get(getVerifyTokenKey)
	if err != nil {
		return "", err
	}

	var gvt egs_integration.GetVerifyTokenResponse
	if err = json.UnmarshalRead(rcVerifyToken, &gvt); err != nil {
		return "", err
	}

	return gvt.Token, nil
}

func epicGamesVerifyToken(accessToken string) error {

	gvta := nod.Begin("verifying EGS token...")
	defer gvta.Done()

	client, err := epicGamesGetClient()
	if err != nil {
		return err
	}

	egsVerifyTokenDir := data.Pwd.AbsRelDirPath(data.EgsVerifyToken, data.Metadata)
	kvVerifyToken, err := kevlar.New(egsVerifyTokenDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if accessToken == "" {
		if accessToken, err = epicGamesGetStoredVerifyToken(); err != nil {
			return err
		}
	}

	if accessToken == "" {
		return errors.New("empty access token, re-connect EGS")
	}

	var vtr *egs_integration.GetVerifyTokenResponse
	vtr, err = egs_integration.GetVerifyToken(accessToken, client)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err = json.MarshalWrite(buf, &vtr); err != nil {
		return err
	}

	return kvVerifyToken.Set(getVerifyTokenKey, buf)
}

func epicGamesResetConnection() error {

	egrc := nod.Begin("resetting EGS connection...")
	defer egrc.Done()

	egsCookieDir := data.Pwd.AbsRelDirPath(data.EgsCookie, data.Metadata)
	egsCookiePath := filepath.Join(egsCookieDir, egsCookiesFilename)

	egsVerifyTokenDir := data.Pwd.AbsRelDirPath(data.EgsVerifyToken, data.Metadata)
	kvVerifyToken, err := kevlar.New(egsVerifyTokenDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if _, err = os.Stat(egsCookiePath); err == nil {
		if err = os.Remove(egsCookiePath); err != nil {
			return nil
		}
	}

	if kvVerifyToken.Has(getVerifyTokenKey) {
		if err = kvVerifyToken.Cut(getVerifyTokenKey); err != nil {
			return err
		}
	}

	return nil
}
