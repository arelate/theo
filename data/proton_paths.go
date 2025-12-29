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

	absUmuRunBinPath := filepath.Join(Pwd.AbsRelDirPath(WineBinaries, Wine), pathways.Sanitize(runtime), latestUmuLauncherVersion, relUmuRunPath)
	if _, err := os.Stat(absUmuRunBinPath); err == nil {
		return absUmuRunBinPath, nil
	}

	return "", os.ErrNotExist
}

func ProtonGeLatestReleasePath(rdx redux.Readable) (string, error) {

	runtime := wine_integration.ProtonGe

	if err := rdx.MustHave(WineBinariesVersionsProperty); err != nil {
		return "", err
	}

	var latestProtonGeVersion string
	if lpgv, ok := rdx.GetLastVal(WineBinariesVersionsProperty, runtime); ok {
		latestProtonGeVersion = lpgv
	}

	if latestProtonGeVersion == "" {
		return "", errors.New("proton-ge version not found, please run setup-wine")
	}

	absProtonGePath := filepath.Join(Pwd.AbsRelDirPath(WineBinaries, Wine), pathways.Sanitize(runtime), latestProtonGeVersion, latestProtonGeVersion)
	if _, err := os.Stat(absProtonGePath); err == nil {
		return absProtonGePath, nil
	}

	return "", os.ErrNotExist
}

func AbsReduxDir() string {
	return Pwd.AbsRelDirPath(Redux, Metadata)
}
