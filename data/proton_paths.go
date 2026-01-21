package data

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"

	"github.com/arelate/southern_light/wine_integration"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

const relUmuRunPath = "umu/umu-run"

func UmuRunLatestReleasePath(rdx redux.Readable) (string, error) {

	runtime := wine_integration.UmuLauncher

	if err := rdx.MustHave(WineBinariesVersionsProperty); err != nil {
		return "", err
	}

	var latestUmuLauncherVersion string
	if lulv, ok := rdx.GetLastVal(WineBinariesVersionsProperty, runtime); ok {
		latestUmuLauncherVersion = lulv
	}

	if latestUmuLauncherVersion == "" {
		return "", errors.New("umu-launcher version not found, please run setup-wine")
	}

	absUmuRunBinPath := filepath.Join(Pwd.AbsRelDirPath(BinUnpacks, Wine), pathways.Sanitize(runtime), latestUmuLauncherVersion, relUmuRunPath)
	if _, err := os.Stat(absUmuRunBinPath); err == nil {
		return absUmuRunBinPath, nil
	}

	return "", os.ErrNotExist
}

func ProtonLatestReleasePath(runtime string, rdx redux.Readable) (string, error) {

	if runtime == "" {
		runtime = wine_integration.ProtonGe
	}

	if err := rdx.MustHave(WineBinariesVersionsProperty); err != nil {
		return "", err
	}

	var latestProtonVersion string
	if lpv, ok := rdx.GetLastVal(WineBinariesVersionsProperty, runtime); ok {
		latestProtonVersion = lpv
	}

	if latestProtonVersion == "" {
		return "", errors.New("proton-ge version not found, please run setup-wine")
	}

	absProtonPath := filepath.Join(Pwd.AbsRelDirPath(BinUnpacks, Wine), pathways.Sanitize(runtime), latestProtonVersion, latestProtonVersion)
	if _, err := os.Stat(absProtonPath); err == nil {
		return absProtonPath, nil
	}

	return "", os.ErrNotExist
}

func AbsReduxDir() string {
	return Pwd.AbsRelDirPath(Redux, Metadata)
}
