package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

func ResetVangoghConnectionHandler(_ *url.URL) error {
	return ResetVangoghConnection()
}

func ResetVangoghConnection() error {
	rsa := nod.Begin("resetting vangogh connection setup...")
	defer rsa.EndWithResult("done, run setup to init")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return rsa.EndWithError(err)
	}

	rdx, err := redux.NewWriter(reduxDir, data.SetupProperties)
	if err != nil {
		return rsa.EndWithError(err)
	}

	setupProperties := []string{
		data.VangoghProtocolProperty,
		data.VangoghAddressProperty,
		data.VangoghPortProperty,
		data.VangoghUsernameProperty,
		data.VangoghPasswordProperty,
	}

	if err := rdx.CutKeys(data.SetupProperties, setupProperties...); err != nil {
		return rsa.EndWithError(err)
	}

	return nil
}
