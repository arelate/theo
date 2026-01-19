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
	innoSetupDirArgTemplate       = "/DIR={dir}"
)

const prefixRelDriveCDir = "drive_c"

func prefixInit(id string, rdx redux.Readable, verbose bool) error {

	cpa := nod.Begin("initializing prefix for %s...", id)
	defer cpa.Done()

	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		return macOsInitPrefix(id, rdx, verbose)
	case vangogh_integration.Linux:
		return linuxInitPrefix(id, rdx, verbose)
	default:
		return data.CurrentOs().ErrUnsupported()
	}
}

func prefixUnpackInstaller(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Writeable) error {

	currentOs := data.CurrentOs()

	puia := nod.Begin(" unpacking %s installers for %s-%s...", id, vangogh_integration.Windows, ii.LangCode)
	defer puia.Done()

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

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

		dstDir := filepath.Join("C:\\Temp", id, link.LocalFilename)
		innoSetupDirArg := strings.Replace(innoSetupDirArgTemplate, "{dir}", dstDir, 1)

		et := &execTask{
			exe:     absInstallerPath,
			workDir: downloadsDir,
			args: []string{
				innoSetupVerySilentArg,
				innoSetupNoRestartArg,
				innoSetupCloseApplicationsArg,
				innoSetupDirArg},
			env:     ii.Env,
			verbose: ii.verbose,
		}

		if err := currentOsWineRun(id, rdx, et, ii.force); err != nil {
			return err
		}
	}

	return nil
}

func prefixPlaceUnpackedFiles(id string, dls vangogh_integration.ProductDownloadLinks, rdx redux.Readable, unpackDir string) error {

	pufa := nod.Begin(" placing unpacked files for %s...", id)
	defer pufa.Done()

	for _, link := range dls {

		if filepath.Ext(link.LocalFilename) != exeExt {
			continue
		}

		absUnpackedPath := filepath.Join(unpackDir, link.LocalFilename)
		if _, err := os.Stat(absUnpackedPath); os.IsNotExist(err) {
			return ErrMissingExtractedPayload
		}

		installedAppPath, err := osInstalledPath(id, vangogh_integration.Windows, link.LanguageCode, rdx)

		if err = placeUnpackedLinkPayload(&link, absUnpackedPath, installedAppPath); err != nil {
			return err
		}
	}

	return nil
}

func prefixGetExePath(id, langCode string, rdx redux.Readable) (string, error) {

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return "", err
	}

	if ep, ok := rdx.GetLastVal(data.PrefixExeProperty, path.Join(prefixName, langCode)); ok && ep != "" {
		absPrefixDir, err := data.AbsPrefixDir(id, rdx)
		if err != nil {
			return "", err
		}

		return filepath.Join(absPrefixDir, prefixRelDriveCDir, ep), nil
	}

	exePath, err := prefixFindGogGameInfoPrimaryPlayTaskExe(id, rdx)
	if err != nil {
		return "", err
	}

	if exePath == "" {
		exePath, err = prefixFindGogGamesLnk(id, rdx)
		if err != nil {
			return "", err
		}
	}

	return exePath, nil
}

func prefixFindGlobFile(id string, rdx redux.Readable, globPattern string) (string, error) {

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", nil
	}

	absPrefixDir, err := data.AbsPrefixDir(id, rdx)
	if err != nil {
		return "", err
	}

	absPrefixDriveCDir := filepath.Join(absPrefixDir, prefixRelDriveCDir)

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

func prefixFindGogGameInstallPath(id string, rdx redux.Readable) (string, error) {
	fi := nod.Begin(" finding install path...")
	defer fi.Done()

	return prefixFindGlobFile(id, rdx, gogGameInstallDir)
}

func prefixFindGogGameInfo(id string, rdx redux.Readable) (string, error) {
	fpggi := nod.Begin(" finding goggame-%s.info...", id)
	defer fpggi.Done()

	return prefixFindGlobFile(id, rdx, strings.Replace(gogGameInfoGlobTemplate, "{id}", id, -1))
}

func prefixFindGogGamesLnk(id string, rdx redux.Readable) (string, error) {
	fpl := nod.Begin(" finding .lnk...")
	defer fpl.Done()

	return prefixFindGlobFile(id, rdx, gogGameLnkGlob)
}

func prefixFindGogGameInfoPrimaryPlayTaskExe(id string, rdx redux.Readable) (string, error) {

	absGogGameInfoPath, err := prefixFindGogGameInfo(id, rdx)
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

func prefixReveal(id string, langCode string) error {

	rpa := nod.Begin("revealing prefix for %s...", id)
	defer rpa.Done()

	rdx, err := redux.NewReader(data.AbsReduxDir(), vangogh_integration.SlugProperty)
	if err != nil {
		return err
	}

	absPrefixDir, err := data.AbsPrefixDir(id, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absPrefixDir); os.IsNotExist(err) {
		rpa.EndWithResult("not found")
		return nil
	}

	absPrefixDriveCPath := filepath.Join(absPrefixDir, prefixRelDriveCDir, gogGamesDir)

	return currentOsReveal(absPrefixDriveCPath)
}
