package data

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/boggydigital/redux"
)

func ServerUrl(rdx redux.Readable, path string, params map[string]string) (*url.URL, error) {
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

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}

	return &url.URL{
		Scheme:   protocol,
		Host:     address,
		Path:     path,
		RawQuery: q.Encode(),
	}, nil
}

func ServerRequest(method, path string, params map[string]string, rdx redux.Readable) (*http.Request, error) {
	protocol := "https"
	var address string

	if err := rdx.MustHave(ServerConnectionProperties); err != nil {
		return nil, err
	}

	if protoVal, ok := rdx.GetLastVal(ServerConnectionProperties, ServerProtocolProperty); ok && protoVal != "" {
		protocol = protoVal
	}

	if addrVal, ok := rdx.GetLastVal(ServerConnectionProperties, ServerAddressProperty); ok && addrVal != "" {
		address = addrVal
	} else {
		return nil, errors.New("address is empty, run connect to setup")
	}

	if portVal, ok := rdx.GetLastVal(ServerConnectionProperties, ServerPortProperty); ok && portVal != "" {
		address += ":" + portVal
	}

	var sessionToken string
	if st, ok := rdx.GetLastVal(ServerConnectionProperties, ServerSessionToken); ok && st != "" {
		sessionToken = st
	}

	u := &url.URL{
		Scheme: protocol,
		Host:   address,
		Path:   path,
	}

	var bodyReader io.Reader

	switch method {
	case http.MethodGet:
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	case http.MethodPost:
		var bodyStrings []string
		for k, v := range params {
			bodyStrings = append(bodyStrings, k+"="+v)
		}
		if len(bodyStrings) > 0 {
			bodyReader = strings.NewReader(strings.Join(bodyStrings, "\n"))
		}
	default:
		return nil, errors.New("method not supported: " + method)
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	if sessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+sessionToken)
	}

	return req, nil
}
