package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func ResetSetupHandler(_ *url.URL) error {
	return ResetSetup()
}

func ResetSetup() error {
	rsa := nod.Begin("resetting theo setup...")
	defer rsa.End()

	rdp, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return rsa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(rdp, data.SetupProperties)
	if err != nil {
		return rsa.EndWithError(err)
	}

	setupProperties := []string{
		data.VangoghProtocolProperty,
		data.VangoghAddressProperty,
		data.VangoghPortProperty,
		data.VangoghUsernameProperty,
		data.VangoghPasswordProperty,
		data.InstallationPathProperty,
	}

	if err := rdx.CutKeys(data.SetupProperties, setupProperties...); err != nil {
		return rsa.EndWithError(err)
	}

	rsa.EndWithResult("done, run setup to init")

	return nil
}
