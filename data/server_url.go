package data

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/boggydigital/redux"
)

func VangoghUrl(path string, data url.Values, rdx redux.Readable) (*url.URL, error) {

	if err := rdx.MustHave(VangoghProperties()...); err != nil {
		return nil, err
	}

	var vangoghUrlStr string
	if vus, ok := rdx.GetLastVal(VangoghUrlProperty, VangoghUrlProperty); ok && vus != "" {
		vangoghUrlStr = vus
	} else {
		return nil, errors.New("vangogh url not set")
	}

	u, err := url.Parse(vangoghUrlStr)
	if err != nil {
		return nil, err
	}

	u.Path = path

	if len(data) > 0 {
		u.RawQuery = data.Encode()
	}

	return u, nil
}

func VangoghRequest(method, path string, data url.Values, rdx redux.Readable) (*http.Request, error) {

	u, err := VangoghUrl(path, data, rdx)
	if err != nil {
		return nil, err
	}

	var sessionToken string
	if st, ok := rdx.GetLastVal(VangoghSessionTokenProperty, VangoghSessionTokenProperty); ok && st != "" {
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
