package data

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/boggydigital/redux"
)

func ServerUrl(path string, data url.Values, rdx redux.Readable) (*url.URL, error) {
	protocol := "https"
	address := ""

	if err := rdx.MustHave(ServerConnectionProperties); err != nil {
		return nil, err
	}

	if protoVal, ok := rdx.GetLastVal(ServerConnectionProperties, ServerProtocolProperty); ok && protoVal != "" {
		protocol = protoVal
	}

	if addrVal, ok := rdx.GetLastVal(ServerConnectionProperties, ServerAddressProperty); ok && addrVal != "" {
		address = addrVal
	} else {
		return nil, errors.New("address is empty, check server connection setup")
	}

	if portVal, ok := rdx.GetLastVal(ServerConnectionProperties, ServerPortProperty); ok && portVal != "" {
		address += ":" + portVal
	}

	u := &url.URL{
		Scheme: protocol,
		Host:   address,
		Path:   path,
	}

	if len(data) > 0 {
		u.RawQuery = data.Encode()
	}

	return u, nil
}

func ServerRequest(method, path string, data url.Values, rdx redux.Readable) (*http.Request, error) {

	u, err := ServerUrl(path, data, rdx)
	if err != nil {
		return nil, err
	}

	var sessionToken string
	if st, ok := rdx.GetLastVal(ServerConnectionProperties, ServerSessionToken); ok && st != "" {
		sessionToken = st
	}

	var bodyReader io.Reader

	if method == http.MethodPost && len(data) > 0 {
		bodyReader = strings.NewReader(data.Encode())
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	if sessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+sessionToken)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}
