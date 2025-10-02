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
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

func ConnectHandler(u *url.URL) error {

	q := u.Query()

	protocol := q.Get("protocol")
	address := q.Get("address")
	port := q.Get("port")

	username := q.Get("username")
	password := q.Get("password")

	reset := q.Has("reset")

	return Connect(protocol, address, port, username, password, reset)
}

func Connect(
	protocol, address, port string,
	username, password string,
	reset bool) error {

	sa := nod.Begin("connecting to the server...")
	defer sa.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.ServerConnectionProperties)
	if err != nil {
		return err
	}

	if reset {
		if err = resetServerConnection(rdx); err != nil {
			return err
		}
	}

	connectionProperties := make(map[string][]string)

	if protocol != "" {
		connectionProperties[data.ServerProtocolProperty] = []string{protocol}
	}

	if address != "" {
		connectionProperties[data.ServerAddressProperty] = []string{address}
	}

	if port != "" {
		connectionProperties[data.ServerPortProperty] = []string{port}
	}

	if username != "" {
		connectionProperties[data.ServerUsernameProperty] = []string{username}
	}

	if len(connectionProperties) > 0 {
		if err = rdx.BatchReplaceValues(data.ServerConnectionProperties, connectionProperties); err != nil {
			return err
		}
	}

	if err = updateSessionToken(password, rdx); err != nil {
		return err
	}

	if err = validateSessionToken(rdx); err != nil {
		return err
	}

	return nil
}

func resetServerConnection(rdx redux.Writeable) error {
	rsa := nod.Begin("resetting server connection...")
	defer rsa.Done()

	if err := rdx.MustHave(data.ServerConnectionProperties); err != nil {
		return err
	}

	setupProperties := []string{
		data.ServerProtocolProperty,
		data.ServerAddressProperty,
		data.ServerPortProperty,
		data.ServerUsernameProperty,
		data.ServerSessionToken,
	}

	if err := rdx.CutKeys(data.ServerConnectionProperties, setupProperties...); err != nil {
		return err
	}

	return nil
}

func validateSessionToken(rdx redux.Readable) error {

	tsa := nod.Begin("validating session token...")
	defer tsa.Done()

	if err := rdx.MustHave(data.ServerConnectionProperties); err != nil {
		return err
	}

	req, err := data.ServerRequest(http.MethodGet, data.ApiAuthSessionPath, nil, rdx)
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

	utcNow := time.Now().UTC()

	if utcNow.Before(ste.Expires.Add(-1 * time.Hour * 24)) {
		tsa.EndWithResult("session is valid")
		return nil
	} else {
		return errors.New("session expires soon, run connect to update")
	}

}

func updateSessionToken(password string, rdx redux.Writeable) error {
	rsa := nod.Begin("updating session token...")
	defer rsa.Done()

	if err := rdx.MustHave(data.ServerConnectionProperties); err != nil {
		return err
	}

	authUrl, err := data.ServerUrl(rdx, data.ApiAuthUserPath, nil)
	if err != nil {
		return err
	}

	var username string
	if up, ok := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerUsernameProperty); ok && up != "" {
		username = up
	} else {
		return errors.New("username not found")
	}

	postData := url.Values{
		"username": {username},
		"password": {password},
	}

	resp, err := http.PostForm(authUrl.String(), postData)
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

	if err = rdx.ReplaceValues(data.ServerConnectionProperties, data.ServerSessionToken, ste.Token); err != nil {
		return err
	}

	if err = rdx.ReplaceValues(data.ServerConnectionProperties, data.ServerSessionExpires, ste.Expires.Format(http.TimeFormat)); err != nil {
		return err
	}

	return nil
}
