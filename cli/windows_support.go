package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/redux"
)

func windowsInstallProduct(id string,
	productDetails *vangogh_integration.ProductDetails,
	link *vangogh_integration.ProductDownloadLink,
	rdx redux.Writeable) error {
	return errors.New("support for Windows installation is not implemented")
}

func windowsReveal(path string) error {
	return errors.New("support for Windows reveal is not implemented")
}

func windowsExecute(path string, et *execTask) error {
	return errors.New("support for Windows execution is not implemented")
}

func windowsUninstallProduct(id, langCode string, rdx redux.Readable) error {
	return errors.New("support for Windows uninstallation is not implemented")
}

func windowsFreeSpace(path string) (int64, error) {
	return -1, errors.New("support for Windows free space determination is not implemented")
}
