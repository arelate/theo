package cli

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/arelate/theo/data"
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

//func testConnection(password string) error {
//
//	tsa := nod.Begin("testing server connection...")
//	defer tsa.EndWithResult("success - server setup is valid")
//
//	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
//	if err != nil {
//		return err
//	}
//
//	rdx, err := redux.NewReader(reduxDir, data.ServerConnectionProperties)
//	if err != nil {
//		return err
//	}
//
//	//if err = testServerConnectivity(rdx); err != nil {
//	//	return err
//	//}
//
//	return nil
//}

//func testServerConnectivity(rdx redux.Readable) error {
//
//	testUrl, err := data.ServerUrl(rdx, data.ApiHealthPath, nil)
//	if err != nil {
//		return err
//	}
//
//	tvaa := nod.Begin(" testing auth for %s...", testUrl.String())
//	defer tvaa.Done()
//
//	req, err := http.NewRequest(http.MethodGet, testUrl.String(), nil)
//	if err != nil {
//		return err
//	}
//
//	if username, ok := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerUsernameProperty); ok && username != "" {
//		if password, sure := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerPasswordProperty); sure && password != "" {
//			req.SetBasicAuth(username, password)
//		} else {
//			return errors.New("password cannot be empty")
//		}
//	} else {
//		return errors.New("username cannot be empty")
//	}
//
//	resp, err := http.DefaultClient.Do(req)
//	if err != nil {
//		return err
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode < 200 || resp.StatusCode > 299 {
//		return errors.New(resp.Status)
//	}
//
//	bts, err := io.ReadAll(resp.Body)
//	if err != nil {
//		return err
//	}
//
//	if string(bts) != "ok" {
//		return errors.New("unexpected health-auth response")
//	}
//
//	tvaa.EndWithResult("done, healthy")
//
//	return nil
//}

func updateSessionToken(password string, rdx redux.Writeable) error {
	rsa := nod.Begin("updating session token...")
	defer rsa.Done()

	if err := rdx.MustHave(data.ServerConnectionProperties); err != nil {
		return err
	}

	authUrl, err := data.ServerUrl(rdx, data.ApiAuthPath, nil)
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

	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, resp.Body); err != nil {
		return err
	}

	return rdx.ReplaceValues(data.ServerConnectionProperties, data.ServerSessionToken, buf.String())
}
