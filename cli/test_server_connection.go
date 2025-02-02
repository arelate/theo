package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"net/http"
	"net/url"
)

func TestServerConnectionHandler(_ *url.URL) error {
	return TestServerConnection()
}

func TestServerConnection() error {

	tsa := nod.Begin("testing server connection...")
	defer tsa.EndWithResult("success - server setup is valid")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return tsa.EndWithError(err)
	}

	rdx, err := redux.NewReader(reduxDir, data.ServerConnectionProperties)
	if err != nil {
		return tsa.EndWithError(err)
	}

	if err := testServerConnectivity(rdx); err != nil {
		return tsa.EndWithError(err)
	}

	if err := testServerAuth(rdx); err != nil {
		return tsa.EndWithError(err)
	}

	return nil
}

func testServerConnectivity(rdx redux.Readable) error {

	testUrl, err := data.ServerUrl(rdx, data.ServerHealthPath, nil)
	if err != nil {
		return err
	}

	tvca := nod.Begin(" testing connectivity to %s...", testUrl.String())
	defer tvca.End()

	resp, err := http.DefaultClient.Get(testUrl.String())
	if err != nil {
		return tvca.EndWithError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return tvca.EndWithError(errors.New(resp.Status))
	}

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return tvca.EndWithError(err)
	}

	if string(bts) != "ok" {
		return tvca.EndWithError(errors.New("unexpected health response"))
	}

	tvca.EndWithResult("done, healthy")

	return nil
}

func testServerAuth(rdx redux.Readable) error {

	testUrl, err := data.ServerUrl(rdx, data.ServerHealthAuthPath, nil)
	if err != nil {
		return err
	}

	tvaa := nod.Begin(" testing auth for %s...", testUrl.String())
	defer tvaa.End()

	req, err := http.NewRequest(http.MethodGet, testUrl.String(), nil)
	if err != nil {
		return tvaa.EndWithError(err)
	}

	if username, ok := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerUsernameProperty); ok && username != "" {
		if password, sure := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerPasswordProperty); sure && password != "" {
			req.SetBasicAuth(username, password)
		} else {
			return tvaa.EndWithError(errors.New("password cannot be empty"))
		}
	} else {
		return tvaa.EndWithError(errors.New("username cannot be empty"))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return tvaa.EndWithError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return tvaa.EndWithError(errors.New(resp.Status))
	}

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return tvaa.EndWithError(err)
	}

	if string(bts) != "ok" {
		return tvaa.EndWithError(errors.New("unexpected health-auth response"))
	}

	tvaa.EndWithResult("done, healthy")

	return nil
}
