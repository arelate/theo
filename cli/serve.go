package cli

import (
	"fmt"
	"github.com/arelate/theo/rest"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"net/http"
	"net/url"
	"strconv"
)

const (
	// https://en.wikipedia.org/wiki/Theo_van_Gogh_(art_dealer)
	defaultPort = 1857
)

func ServeHandler(u *url.URL) error {

	port := defaultPort
	if portStr := vangogh_local_data.ValueFromUrl(u, "port"); portStr != "" {
		if portNum, err := strconv.Atoi(portStr); err == nil {
			port = portNum
		}
	}

	stderr := u.Query().Has("stderr")

	return Serve(port, stderr)
}

func Serve(port int, stderr bool) error {

	if stderr {
		nod.EnableStdErrLogger()
		nod.DisableOutput(nod.StdOut)
	}

	if err := rest.Init(); err != nil {
		return err
	}

	rest.HandleFuncs()

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
