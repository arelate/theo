package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

type SetupProperties map[string]string

func SetVangoghConnectionHandler(u *url.URL) error {

	q := u.Query()

	protocol := q.Get("protocol")
	address := q.Get("address")
	port := q.Get("port")

	username := q.Get("username")
	password := q.Get("password")

	return SetVangoghConnection(protocol, address, port, username, password)
}

func SetVangoghConnection(
	protocol, address, port string,
	username, password string) error {

	// resetting setup properties since not every property is required (e.g. port, protocol)
	// and it would be possible to end up with a set of properties that will let to failures
	// in non-obvious ways
	if err := ResetVangoghConnection(); err != nil {
		return err
	}

	sa := nod.Begin("setting up vangogh connection...")
	defer sa.EndWithResult("done, run test-setup to validate")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return sa.EndWithError(err)
	}

	rdx, err := redux.NewWriter(reduxDir, data.SetupProperties)
	if err != nil {
		return sa.EndWithError(err)
	}

	setupProperties := make(map[string][]string)

	if protocol != "" {
		setupProperties[data.VangoghProtocolProperty] = []string{protocol}
	}

	setupProperties[data.VangoghAddressProperty] = []string{address}

	if port != "" {
		setupProperties[data.VangoghPortProperty] = []string{port}
	}

	setupProperties[data.VangoghUsernameProperty] = []string{username}
	setupProperties[data.VangoghPasswordProperty] = []string{password}

	if err := rdx.BatchReplaceValues(data.SetupProperties, setupProperties); err != nil {
		return sa.EndWithError(err)
	}

	return nil
}
