package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

func ResetServerConnectionHandler(_ *url.URL) error {
	return ResetServerConnection()
}

func ResetServerConnection() error {
	rsa := nod.Begin("resetting server connection setup...")
	defer rsa.EndWithResult("done, run setup to init")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return rsa.EndWithError(err)
	}

	rdx, err := redux.NewWriter(reduxDir, data.ServerConnectionProperties)
	if err != nil {
		return rsa.EndWithError(err)
	}

	setupProperties := []string{
		data.ServerProtocolProperty,
		data.ServerAddressProperty,
		data.ServerPortProperty,
		data.ServerUsernameProperty,
		data.ServerPasswordProperty,
	}

	if err := rdx.CutKeys(data.ServerConnectionProperties, setupProperties...); err != nil {
		return rsa.EndWithError(err)
	}

	return nil
}
