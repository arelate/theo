package cli

import (
	"bytes"
	_ "embed"
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	dxVkMod = "dxvk"

	// https://gitlab.winehq.org/wine/wine/-/wikis/Commands/winecfg#screen-resolution-dpi-setting
	retinaMod = "retina"
)

const (
	retinaOnFilename  = "retina_on.reg"
	retinaOffFilename = "retina_off.reg"
	dxVkDlls64Glob    = "x64/*.dll"
	dxVkDlls32Glob    = "x32/*.dll"
	pfxSystem64Path   = "windows/system32"
	pfxSystem32Path   = "windows/syswow64"
)

var (
	//go:embed "pfx_reg/retina_on.reg"
	retinaOnReg []byte
	//go:embed "pfx_reg/retina_off.reg"
	retinaOffReg []byte
)

func ModPrefixHandler(u *url.URL) error {

	q := u.Query()

	name := q.Get("name")

	releaseSelector := data.ReleaseSelectorFromUrl(u)

	var on []string
	var off []string
	if q.Has("on") {
		on = strings.Split(q.Get("on"), ",")
	}
	if q.Has("off") {
		off = strings.Split(q.Get("off"), ",")
	}
	force := q.Has("force")

	return ModPrefix(name, releaseSelector, on, off, force)
}

func ModPrefix(name string, releaseSelector *data.GitHubReleaseSelector, on, off []string, force bool) error {

	mpa := nod.Begin("modding prefix %s...", name)
	defer mpa.EndWithResult("done")

	PrintReleaseSelector([]vangogh_local_data.OperatingSystem{CurrentOS()}, releaseSelector)

	if CurrentOS() != vangogh_local_data.MacOS {
		mpa.EndWithResult("prefix modification are currently only applicable to macOS")
		return nil
	}

	if releaseSelector == nil {
		releaseSelector = &data.GitHubReleaseSelector{}
	}

	if releaseSelector.Owner == "" && releaseSelector.Repo == "" {
		dws, err := data.GetDefaultWineSource(CurrentOS())
		if err != nil {
			return mpa.EndWithError(err)
		}
		releaseSelector.Owner = dws.Owner
		releaseSelector.Repo = dws.Repo
	}

	absWineBin, err := data.GetWineBinary(CurrentOS(), releaseSelector)
	if err != nil {
		return mpa.EndWithError(err)
	}

	if _, err := os.Stat(absWineBin); err != nil {
		return mpa.EndWithError(err)
	}

	absPrefixDir, err := data.GetAbsPrefixDir(name)
	if err != nil {
		return mpa.EndWithError(err)
	}

	wineCtx := &data.WineContext{
		BinPath:    absWineBin,
		PrefixPath: absPrefixDir,
	}

	toggles := make(map[string]bool)

	for _, feat := range off {
		toggles[feat] = false
	}
	for _, feat := range on {
		toggles[feat] = true
	}

	return toggleFeatures(name, toggles, wineCtx, force)
}

func toggleFeatures(pfxName string, toggles map[string]bool, wineCtx *data.WineContext, force bool) error {

	tmpa := nod.Begin(" toggling features for prefix %s...", pfxName)
	defer tmpa.EndWithResult("done")

	if len(toggles) == 0 {
		tmpa.EndWithResult("no features provided")
		return nil
	}

	for feat, flag := range toggles {
		switch feat {
		case retinaMod:
			if err := toggleRetina(wineCtx, flag, force); err != nil {
				return tmpa.EndWithError(err)
			}
		case dxVkMod:
			if err := toggleDxVk(wineCtx, flag, force); err != nil {
				return tmpa.EndWithError(err)
			}
		default:
			return tmpa.EndWithError(errors.New("unknown prefix modification feature: " + feat))
		}
	}

	return nil
}

func toggleRetina(wineCtx *data.WineContext, flag bool, force bool) error {

	absDriveCroot := filepath.Join(wineCtx.PrefixPath, driveCpath)

	regFilename := retinaOffFilename
	regContent := retinaOffReg
	if flag {
		regFilename = retinaOnFilename
		regContent = retinaOnReg
	}

	absRegPath := filepath.Join(absDriveCroot, regFilename)
	if _, err := os.Stat(absRegPath); os.IsNotExist(err) || (err == nil && force) {
		if err := createRegFile(absRegPath, regContent); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return data.RegeditWinePrefix(wineCtx, absRegPath)
}

func createRegFile(absPath string, content []byte) error {

	regFile, err := os.Create(absPath)
	if err != nil {
		return err
	}
	defer regFile.Close()

	if _, err := io.Copy(regFile, bytes.NewReader(content)); err != nil {
		return err
	}

	return nil
}

func toggleDxVk(wineCtx *data.WineContext, flag bool, force bool) error {

	// check release binaries for DXVK-macos
	// based on the flag copy over / restore .dll files

	return nil
}
