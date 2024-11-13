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

	mdp, err := pathways.GetAbsDir(data.Metadata)
	if err != nil {
		return tsa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(mdp, data.SetupProperties)
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

func vangoghUrl(rdx kevlar.ReadableRedux) (*url.URL, error) {
	protocol := "https"
	address := ""

	if protoVal, ok := rdx.GetLastVal(data.SetupProperties, data.VangoghProtocolProperty); ok && protoVal != "" {
		protocol = protoVal
	}

	if addrVal, ok := rdx.GetLastVal(data.SetupProperties, data.VangoghAddressProperty); ok && addrVal != "" {
		address = addrVal
	} else {
		return nil, errors.New("vangogh address cannot be empty")
	}

	if portVal, ok := rdx.GetLastVal(data.SetupProperties, data.VangoghPortProperty); ok && portVal != "" {
		address += ":" + portVal
	}

	return &url.URL{
		Scheme: protocol,
		Host:   address,
		Path:   "/health",
	}, nil
}

func testVangoghConnectivity(rdx kevlar.ReadableRedux) error {

	tvca := nod.Begin(" testing vangogh connectivity...")
	defer tvca.End()

	testUrl, err := vangoghUrl(rdx)
	if err != nil {
		return tvca.EndWithError(err)
	}

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
		return tvca.EndWithError(errors.New("unexpected vangogh health response"))
	}

	tvca.EndWithResult("done, healthy")

	return nil
}

func testVangoghAuth(rdx kevlar.ReadableRedux) error {
	tvaa := nod.Begin(" testing vangogh username/password...")
	defer tvaa.End()

	testUrl, err := vangoghUrl(rdx)
	if err != nil {
		return tvaa.EndWithError(err)
	}

	req, err := http.NewRequest(http.MethodGet, testUrl.String(), nil)
	if err != nil {
		return tvaa.EndWithError(err)
	}

	if username, ok := rdx.GetLastVal(data.SetupProperties, data.VangoghUsernameProperty); ok && username != "" {
		if password, sure := rdx.GetLastVal(data.SetupProperties, data.VangoghPasswordProperty); sure && password != "" {
			req.SetBasicAuth(username, password)
		} else {
			return tvaa.EndWithError(errors.New("vangogh password cannot be empty"))
		}
	} else {
		return tvaa.EndWithError(errors.New("vangogh username cannot be empty"))
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
		return tvaa.EndWithError(errors.New("unexpected vangogh health-auth response"))
	}

	tvaa.EndWithResult("done, healthy")

	return nil
}

func testInstallationPath(rdx kevlar.ReadableRedux) error {

	tipa := nod.Begin(" testing installation path...")
	defer tipa.End()

	if ip, ok := rdx.GetLastVal(data.SetupProperties, data.InstallationPathProperty); ok && ip != "" {
		if _, err := os.Stat(ip); err != nil {
			return tipa.EndWithError(err)
		}
	}

	tipa.EndWithResult("default installation path will be used")

	return nil
}
