package data

import (
	_ "embed"
	"errors"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"os"
	"path/filepath"
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

	wineBinaries, err := pathways.GetAbsRelDir(WineBinaries)
	if err != nil {
		return "", err
	}

	absUmuRunBinPath := filepath.Join(wineBinaries, busan.Sanitize(runtime), latestUmuLauncherVersion, relUmuRunPath)
	if _, err = os.Stat(absUmuRunBinPath); err == nil {
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

	wineBinaries, err := pathways.GetAbsRelDir(WineBinaries)
	if err != nil {
		return "", err
	}

	absProtonGePath := filepath.Join(wineBinaries, busan.Sanitize(runtime), latestProtonGeVersion, latestProtonGeVersion)
	if _, err = os.Stat(absProtonGePath); err == nil {
		return absProtonGePath, nil
	}

	return "", os.ErrNotExist
}
