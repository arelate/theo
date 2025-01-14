package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
)

func windowsInstallProduct(id string,
	metadata *vangogh_integration.TheoMetadata,
	link *vangogh_integration.TheoDownloadLink,
	absInstallerPath, installedAppsDir string) error {
	return errors.New("support for Windows installation is not implemented")
}

func windowsReveal(path string) error {
	return errors.New("support for Windows reveal is not implemented")
}

func windowsExecute(path string) error {
	return errors.New("support for Windows execution is not implemented")
}

func windowsUninstallProduct(title, installationDir, langCode, bundleName string) error {
	return errors.New("support for Windows uninstallation is not implemented")
}
