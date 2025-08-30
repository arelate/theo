package data

import (
	"errors"
	"net/url"

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
