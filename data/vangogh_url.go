package data

import (
	"errors"
	"github.com/boggydigital/kevlar"
	"net/url"
)

func VangoghUrl(path string, rdx kevlar.ReadableRedux) (*url.URL, error) {
	protocol := "https"
	address := ""

	if err := rdx.MustHave(SetupProperties); err != nil {
		return nil, err
	}

	if protoVal, ok := rdx.GetLastVal(SetupProperties, VangoghProtocolProperty); ok && protoVal != "" {
		protocol = protoVal
	}

	if addrVal, ok := rdx.GetLastVal(SetupProperties, VangoghAddressProperty); ok && addrVal != "" {
		address = addrVal
	} else {
		return nil, errors.New("address cannot be empty")
	}

	if portVal, ok := rdx.GetLastVal(SetupProperties, VangoghPortProperty); ok && portVal != "" {
		address += ":" + portVal
	}

	return &url.URL{
		Scheme: protocol,
		Host:   address,
		Path:   path,
	}, nil
}
