package cli

import (
	"errors"
	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/redux"
)

const exeExt = ".exe"

func windowsInstallProduct(id string,
	dls vangogh_integration.ProductDownloadLinks,
	rdx redux.Writeable,
	force bool) error {
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

func windowsFindGogGameInfo(id, langCode string, rdx redux.Readable) (string, error) {
	return "", errors.New("support for Windows goggame-{id}.info is not implemented")
}

func windowsFindGogGamesLnk(id, langCode string, rdx redux.Readable) (string, error) {
	return "", errors.New("support for Windows .lnk is not implemented")
}

func windowsExecTaskGogGameInfo(absGogGameInfoPath string, gogGameInfo *gog_integration.GogGameInfo, et *execTask) (*execTask, error) {
	return et, nil
}

func windowsExecTaskLnk(absLnkPath string, et *execTask) (*execTask, error) {
	return et, nil
}
