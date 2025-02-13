package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/redux"
)

func windowsInstallProduct(id string,
	metadata *vangogh_integration.TheoMetadata,
	link *vangogh_integration.TheoDownloadLink,
	rdx redux.Writeable) error {
	return errors.New("support for Windows installation is not implemented")
}

func windowsReveal(path string) error {
	return errors.New("support for Windows reveal is not implemented")
}

func windowsExecute(path string, env []string, verbose bool) error {
	return errors.New("support for Windows execution is not implemented")
}

func windowsUninstallProduct(id, langCode string, rdx redux.Readable) error {
	return errors.New("support for Windows uninstallation is not implemented")
}
