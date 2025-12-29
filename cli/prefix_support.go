package cli

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	innoSetupVerySilentArg        = "/VERYSILENT"
	innoSetupNoRestartArg         = "/NORESTART"
	innoSetupCloseApplicationsArg = "/CLOSEAPPLICATIONS"
)

const relPrefixDriveCDir = "drive_c"

func prefixGetExePath(id, langCode string, rdx redux.Readable) (string, error) {

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return "", err
	}

	if ep, ok := rdx.GetLastVal(data.PrefixExeProperty, path.Join(prefixName, langCode)); ok && ep != "" {
		absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
		if err != nil {
			return "", err
		}

		return filepath.Join(absPrefixDir, relPrefixDriveCDir, ep), nil
	}

	exePath, err := prefixFindGogGameInfoPrimaryPlayTaskExe(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	if exePath == "" {
		exePath, err = prefixFindGogGamesLnk(id, langCode, rdx)
		if err != nil {
			return "", err
		}
	}

	return exePath, nil
}

func prefixFindGlobFile(id, langCode string, rdx redux.Readable, globPattern string) (string, error) {

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	absPrefixDriveCDir := filepath.Join(absPrefixDir, relPrefixDriveCDir)

	matches, err := filepath.Glob(filepath.Join(absPrefixDriveCDir, globPattern))
	if err != nil {
		return "", err
	}

	filteredMatches := make([]string, 0, len(matches))
	for _, match := range matches {
		if _, filename := filepath.Split(match); strings.HasPrefix(filename, ".") {
			continue
		}
		filteredMatches = append(filteredMatches, match)
	}

	if len(filteredMatches) == 1 {

		if _, err = os.Stat(filteredMatches[0]); err == nil {
			return filteredMatches[0], nil
		} else if os.IsNotExist(err) {
			return "", nil
		} else {
			return "", err
		}

	}

	return "", nil
}

func prefixFindGogGameInstallPath(id, langCode string, rdx redux.Readable) (string, error) {
	fi := nod.Begin(" finding install path...")
	defer fi.Done()

	return prefixFindGlobFile(id, langCode, rdx, gogGameInstallDir)
}

func prefixFindGogGameInfo(id, langCode string, rdx redux.Readable) (string, error) {
	fpggi := nod.Begin(" finding goggame-%s.info...", id)
	defer fpggi.Done()

	return prefixFindGlobFile(id, langCode, rdx, strings.Replace(gogGameInfoGlobTemplate, "{id}", id, -1))
}

func prefixFindGogGamesLnk(id, langCode string, rdx redux.Readable) (string, error) {
	fpl := nod.Begin(" finding .lnk...")
	defer fpl.Done()

	return prefixFindGlobFile(id, langCode, rdx, gogGameLnkGlob)
}

func prefixFindGogGameInfoPrimaryPlayTaskExe(id, langCode string, rdx redux.Readable) (string, error) {

	absGogGameInfoPath, err := prefixFindGogGameInfo(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	gogGameInfo, err := gog_integration.GetGogGameInfo(absGogGameInfoPath)
	if err != nil {
		return "", err
	}

	var relExePath string
	ppt, err := gogGameInfo.GetPlayTask("")
	if err != nil {
		return "", err
	}

	relExePath = ppt.Path

	if relExePath == "" {
		return "", errors.New("cannot determine primary or first playTask for " + id)
	}

	absExeDir, _ := filepath.Split(absGogGameInfoPath)

	return filepath.Join(absExeDir, relExePath), nil
}

func prefixInit(id string, langCode string, rdx redux.Readable, verbose bool) error {

	cpa := nod.Begin("initializing prefix for %s...", id)
	defer cpa.Done()

	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		return macOsInitPrefix(id, langCode, rdx, verbose)
	case vangogh_integration.Linux:
		return linuxInitPrefix(id, langCode, rdx, verbose)
	default:
		return data.CurrentOs().ErrUnsupported()
	}
}

func prefixInstallProduct(id string, dls vangogh_integration.ProductDownloadLinks, ii *InstallInfo, rdx redux.Writeable) error {

	currentOs := data.CurrentOs()

	wipa := nod.Begin("installing %s for %s...", id, vangogh_integration.Windows)
	defer wipa.Done()

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)
	installedAppsDir := data.Pwd.AbsDirPath(data.InstalledApps)

	productDetails, err := getProductDetails(id, rdx, ii.force)
	if err != nil {
		return err
	}

	if err = hasFreeSpaceForProduct(productDetails, installedAppsDir, ii, nil); err != nil {
		return err
	}

	var currentOsWineRun wineRunFunc
	switch currentOs {
	case vangogh_integration.MacOS:
		currentOsWineRun = macOsWineRun
	case vangogh_integration.Linux:
		currentOsWineRun = linuxProtonRun
	default:
		return currentOs.ErrUnsupported()
	}

	for _, link := range dls {

		if linkExt := filepath.Ext(link.LocalFilename); linkExt != exeExt {
			continue
		}

		absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

		et := &execTask{
			exe:     absInstallerPath,
			workDir: downloadsDir,
			args:    []string{innoSetupVerySilentArg, innoSetupNoRestartArg, innoSetupCloseApplicationsArg},
			env:     ii.Env,
			verbose: ii.verbose,
		}

		if err = currentOsWineRun(id, ii.LangCode, rdx, et, ii.force); err != nil {
			return err
		}
	}

	return nil
}

func prefixCreateInventory(id, langCode string, rdx redux.Readable, utcTime int64) error {

	cpifma := nod.Begin(" creating installed files inventory...")
	defer cpifma.Done()

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	return createInventory(absPrefixDir, id, langCode, vangogh_integration.Windows, rdx, utcTime)
}

func prefixReveal(id string, langCode string) error {

	rpa := nod.Begin("revealing prefix for %s...", id)
	defer rpa.Done()

	rdx, err := redux.NewReader(data.AbsReduxDir(), vangogh_integration.SlugProperty)
	if err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absPrefixDir); os.IsNotExist(err) {
		rpa.EndWithResult("not found")
		return nil
	}

	absPrefixDriveCPath := filepath.Join(absPrefixDir, relPrefixDriveCDir, gogGamesDir)

	return currentOsReveal(absPrefixDriveCPath)
}
