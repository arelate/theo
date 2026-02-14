package cli

import (
	"os"
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

const lnkGlob = "*.lnk"

const prefixRelDriveCDir = "drive_c"

func prefixInit(id string, rdx redux.Readable, verbose bool) error {

	cpa := nod.Begin("initializing prefix for %s...", id)
	defer cpa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	absPrefixDir, err := data.AbsPrefixDir(id, rdx)
	if err != nil {
		return err
	}

	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		return macOsInitPrefix(absPrefixDir, verbose)
	case vangogh_integration.Linux:
		return linuxInitPrefix(absPrefixDir, verbose)
	default:
		return data.CurrentOs().ErrUnsupported()
	}
}

func steamPrefixInit(steamAppId string, verbose bool) error {

	cpa := nod.Begin("initializing Steam prefix for %s...", steamAppId)
	defer cpa.Done()

	absSteamPrefixDir, err := data.AbsSteamPrefixDir(steamAppId)
	if err != nil {
		return err
	}

	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		return macOsInitPrefix(absSteamPrefixDir, verbose)
	case vangogh_integration.Linux:
		return linuxInitPrefix(absSteamPrefixDir, verbose)
	default:
		return data.CurrentOs().ErrUnsupported()
	}
}

func prefixUnpackInstallers(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Readable, unpackDir string) error {

	currentOs := data.CurrentOs()

	puia := nod.Begin(" unpacking %s installers for %s-%s...", id, vangogh_integration.Windows, ii.LangCode)
	defer puia.Done()

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	var currentOsTaskExec wineTaskExecFunc
	switch currentOs {
	case vangogh_integration.MacOS:
		currentOsTaskExec = macOsWineExecTask
	case vangogh_integration.Linux:
		currentOsTaskExec = linuxProtonExecTask
	default:
		return currentOs.ErrUnsupported()
	}

	for _, link := range dls {

		if !isLinkExecutable(&link, vangogh_integration.Windows) {
			continue
		}

		absDstDir := filepath.Join(unpackDir, link.LocalFilename)
		if _, err := os.Stat(absDstDir); os.IsNotExist(err) {
			if err = os.MkdirAll(absDstDir, 0755); err != nil {
				return err
			}
		}

		absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)
		prefixDstDir := filepath.Join("C:\\Temp", id, link.LocalFilename)

		innoSetupDirArg := strings.Replace(innoSetupDirArgTemplate, "{dir}", prefixDstDir, 1)

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

		if err := currentOsTaskExec(id, et); err != nil {
			return err
		}
	}

	return nil
}

func prefixPlaceUnpackedFiles(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Readable, unpackDir string) error {

	pufa := nod.Begin(" placing unpacked files for %s...", id)
	defer pufa.Done()

	for _, link := range dls {

		if !isLinkExecutable(&link, vangogh_integration.Windows) {
			continue
		}

		absUnpackedPath := filepath.Join(unpackDir, link.LocalFilename)
		if _, err := os.Stat(absUnpackedPath); os.IsNotExist(err) {
			return ErrMissingExtractedPayload
		}

		installedAppPath, err := originOsInstalledPath(id, ii, rdx)

		if err = placeUnpackedLinkPayload(&link, absUnpackedPath, installedAppPath); err != nil {
			return err
		}
	}

	return nil
}

func prefixFindGlobFile(id string, ii *InstallInfo, rdx redux.Readable, globPattern string) (string, error) {

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", nil
	}

	installedAppDir, err := originOsInstalledPath(id, ii, rdx)
	if err != nil {
		return "", err
	}

	matches, err := filepath.Glob(filepath.Join(installedAppDir, globPattern))
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

func prefixFindGogGameInfo(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {
	fpggi := nod.Begin(" finding goggame-%s.info...", id)
	defer fpggi.Done()

	return prefixFindGlobFile(id, ii, rdx, strings.Replace(gog_integration.GogGameInfoFilenameTemplate, "{id}", id, -1))
}

func prefixFindGogGamesLnk(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {
	fpl := nod.Begin(" finding .lnk...")
	defer fpl.Done()

	return prefixFindGlobFile(id, ii, rdx, lnkGlob)
}
