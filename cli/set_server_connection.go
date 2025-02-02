package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

type SetupProperties map[string]string

func SetServerConnectionHandler(u *url.URL) error {

	q := u.Query()

	protocol := q.Get("protocol")
	address := q.Get("address")
	port := q.Get("port")

	username := q.Get("username")
	password := q.Get("password")

	return SetServerConnection(protocol, address, port, username, password)
}

func SetServerConnection(
	protocol, address, port string,
	username, password string) error {

	// resetting setup properties since not every property is required (e.g. port, protocol)
	// and it would be possible to end up with a set of properties that will let to failures
	// in non-obvious ways
	if err := ResetServerConnection(); err != nil {
		return err
	}

	sa := nod.Begin("setting up server connection...")
	defer sa.EndWithResult("done, run test-setup to validate")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return sa.EndWithError(err)
	}

	rdx, err := redux.NewWriter(reduxDir, data.ServerConnectionProperties)
	if err != nil {
		return sa.EndWithError(err)
	}

	setupProperties := make(map[string][]string)

	if protocol != "" {
		setupProperties[data.ServerProtocolProperty] = []string{protocol}
	}

	setupProperties[data.ServerAddressProperty] = []string{address}

	if port != "" {
		setupProperties[data.ServerPortProperty] = []string{port}
	}

	setupProperties[data.ServerUsernameProperty] = []string{username}
	setupProperties[data.ServerPasswordProperty] = []string{password}

	if err := rdx.BatchReplaceValues(data.ServerConnectionProperties, setupProperties); err != nil {
		return sa.EndWithError(err)
	}

	return nil
}
