package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"io"
	"net/http"
	"net/url"
	"os"
)

func TestSetupHandler(_ *url.URL) error {
	return TestSetup()
}

func TestSetup() error {

	tsa := nod.Begin("testing theo setup...")
	defer tsa.End()

	rdp, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return tsa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(rdp, data.SetupProperties)
	if err != nil {
		return tsa.EndWithError(err)
	}

	if err := testVangoghConnectivity(rdx); err != nil {
		return tsa.EndWithError(err)
	}

	if err := testVangoghAuth(rdx); err != nil {
		return tsa.EndWithError(err)
	}

	if err := testInstallationPath(rdx); err != nil {
		return tsa.EndWithError(err)
	}

	tsa.EndWithResult("success - theo setup is valid")

	return nil
}

func testVangoghConnectivity(rdx kevlar.ReadableRedux) error {

	testUrl, err := data.VangoghUrl(rdx, data.VangoghHealthPath, nil)
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

func testVangoghAuth(rdx kevlar.ReadableRedux) error {

	testUrl, err := data.VangoghUrl(rdx, data.VangoghHealthAuthPath, nil)
	if err != nil {
		return err
	}

	tvaa := nod.Begin(" testing auth for %s...", testUrl.String())
	defer tvaa.End()

	req, err := http.NewRequest(http.MethodGet, testUrl.String(), nil)
	if err != nil {
		return tvaa.EndWithError(err)
	}

	if username, ok := rdx.GetLastVal(data.SetupProperties, data.VangoghUsernameProperty); ok && username != "" {
		if password, sure := rdx.GetLastVal(data.SetupProperties, data.VangoghPasswordProperty); sure && password != "" {
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

func testInstallationPath(rdx kevlar.ReadableRedux) error {

	tipa := nod.Begin(" testing installation path validity...")
	defer tipa.End()

	if ip, ok := rdx.GetLastVal(data.SetupProperties, data.InstallationPathProperty); ok && ip != "" {
		if _, err := os.Stat(ip); err != nil {
			return tipa.EndWithError(err)
		}
	}

	tipa.EndWithResult("not set, will use default")

	return nil
}
