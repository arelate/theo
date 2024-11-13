package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func SetupHandler(u *url.URL) error {

	q := u.Query()

	protocol := q.Get("protocol")
	address := q.Get("address")
	port := q.Get("port")

	username := q.Get("username")
	password := q.Get("password")

	installPath := q.Get("installation-path")

	return Setup(protocol, address, port,
		username, password,
		installPath)
}

func Setup(
	protocol, address, port string,
	username, password string,
	installPath string) error {

	sa := nod.Begin("setting up theo...")
	defer sa.End()

	mdp, err := pathways.GetAbsDir(data.Metadata)
	if err != nil {
		return sa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(mdp, data.SetupProperties)
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

	if installPath != "" {
		setupProperties[data.InstallationPathProperty] = []string{installPath}
	}

	if err := rdx.BatchReplaceValues(data.SetupProperties, setupProperties); err != nil {
		return sa.EndWithError(err)
	}

	sa.EndWithResult("done, now you can test-setup")

	return nil
}
