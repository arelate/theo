package cli

import (
	"bufio"
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	innoSetupVerySilentArg        = "/VERYSILENT"
	innoSetupNoRestartArg         = "/NORESTART"
	innoSetupCloseApplicationsArg = "/CLOSEAPPLICATIONS"
)

func WineInstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	_, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	var env []string
	if q.Has("env") {
		env = strings.Split(q.Get("env"), ",")
	}
	keepDownloads := q.Has("keep-downloads")
	noSteamShortcut := q.Has("no-steam-shortcut")
	verbose := q.Has("verbose")
	force := q.Has("force")

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	return WineInstall(langCode, env, downloadTypes, keepDownloads, noSteamShortcut, verbose, force, ids...)
}

func WineInstall(langCode string,
	env []string,
	downloadTypes []vangogh_integration.DownloadType,
	keepDownloads bool,
	noSteamShortcut bool,
	verbose bool,
	force bool,
	ids ...string) error {

	start := time.Now().UTC().Unix()

	wia := nod.Begin("installing %s versions on %s...",
		vangogh_integration.Windows,
		data.CurrentOs())
	defer wia.Done()

	if data.CurrentOs() == vangogh_integration.Windows {
		wia.EndWithResult("WINE install is not required on Windows, use install")
		return nil
	}

	windowsOs := []vangogh_integration.OperatingSystem{vangogh_integration.Windows}
	langCodes := []string{langCode}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty)
	if err != nil {
		return err
	}

	notInstalled, err := wineFilterNotInstalled(langCode, rdx, ids...)
	if err != nil {
		return err
	}

	if len(notInstalled) > 0 {
		if !force {
			ids = notInstalled
		}
	} else if !force {
		wia.EndWithResult("all requested products are already installed")
		return nil
	}

	if err := BackupMetadata(); err != nil {
		return err
	}

	if err = Download(windowsOs, langCodes, downloadTypes, force, ids...); err != nil {
		return err
	}

	if err = Validate(windowsOs, langCodes, downloadTypes, ids...); err != nil {
		return err
	}

	if err = initPrefix(langCode, verbose, rdx, ids...); err != nil {
		return err
	}

	for _, id := range ids {
		if err = wineInstallProduct(id, langCode, rdx, env, downloadTypes, verbose, force); err != nil {
			return err
		}

		if err = createInstalledFilesManifest(id, langCode, rdx, start); err != nil {
			return err
		}
	}

	if err := DefaultPrefixEnv(ids); err != nil {
		return err
	}

	if !noSteamShortcut {
		if err := AddSteamShortcut(langCode, wineRunLaunchOptionsTemplate, force, ids...); err != nil {
			return err
		}
	}

	if !keepDownloads {
		if err = RemoveDownloads(windowsOs, langCodes, downloadTypes, force, ids...); err != nil {
			return err
		}
	}

	if err = pinInstalledMetadata(windowsOs, langCode, force, ids...); err != nil {
		return err
	}

	ip := &installParameters{
		operatingSystem: vangogh_integration.Windows,
		langCode:        langCode,
		downloadTypes:   downloadTypes,
		keepDownloads:   keepDownloads,
		noSteamShortcut: noSteamShortcut,
	}

	if err = pinInstallParameters(ip, ids...); err != nil {
		return err
	}

	if err := RevealPrefix(langCode, ids...); err != nil {
		return err
	}

	return nil
}

func wineFilterNotInstalled(langCode string, rdx redux.Readable, ids ...string) ([]string, error) {

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return nil, err
	}

	notInstalled := make([]string, 0, len(ids))

	for _, id := range ids {

		absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
		if err != nil {
			notInstalled = append(notInstalled, id)
			continue
		}

		absPrefixDriveCDir := filepath.Join(absPrefixDir, relPrefixDriveCDir)

		if _, err := os.Stat(absPrefixDriveCDir); err == nil {
			continue
		}

		notInstalled = append(notInstalled, id)
	}

	return notInstalled, nil
}

func wineInstallProduct(id, langCode string, rdx redux.Readable, env []string, downloadTypes []vangogh_integration.DownloadType, verbose, force bool) error {

	currentOs := data.CurrentOs()

	wipa := nod.Begin("installing %s version on %s...", vangogh_integration.Windows, currentOs)
	defer wipa.Done()

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	metadata, err := getTheoMetadata(id, force)
	if err != nil {
		return err
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(vangogh_integration.Windows).
		FilterLanguageCodes(langCode).
		FilterDownloadTypes(downloadTypes...)

	var currentOsWineRun wineRunFunc
	switch currentOs {
	case vangogh_integration.MacOS:
		currentOsWineRun = macOsWineRun
	case vangogh_integration.Linux:
		currentOsWineRun = linuxProtonRun
	default:
		return errors.New("wine-install: unsupported operating system")
	}

	for _, link := range dls {

		if linkExt := filepath.Ext(link.LocalFilename); linkExt != exeExt {
			continue
		}

		absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

		if err := currentOsWineRun(id, langCode, rdx, env, verbose, force, absInstallerPath,
			innoSetupVerySilentArg, innoSetupNoRestartArg, innoSetupCloseApplicationsArg); err != nil {
			return err
		}
	}

	return nil
}

func initPrefix(langCode string, verbose bool, rdx redux.Readable, ids ...string) error {

	cpa := nod.NewProgress("initializing prefixes for %s...", strings.Join(ids, ","))
	defer cpa.Done()

	cpa.TotalInt(len(ids))

	var currentOsWineInitPrefix wineInitPrefixFunc
	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		currentOsWineInitPrefix = macOsInitPrefix
	case vangogh_integration.Linux:
		currentOsWineInitPrefix = linuxInitPrefix
	default:
		return errors.New("init-prefix: unsupported operating system")
	}

	for _, id := range ids {

		if err := currentOsWineInitPrefix(id, langCode, rdx, verbose); err != nil {
			return err
		}

		cpa.Increment()
	}

	return nil
}

func createInstalledFilesManifest(id, langCode string, rdx redux.Readable, utcTime int64) error {

	eifa := nod.Begin(" creating installed files manifest...")
	defer eifa.Done()

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	relFiles, err := data.GetRelFilesModifiedAfter(absPrefixDir, utcTime)
	if err != nil {
		return err
	}

	return appendManifest(id, langCode, vangogh_integration.Windows, rdx, relFiles...)
}

func readManifest(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) ([]string, error) {
	absManifestFilename, err := data.GetAbsManifestFilename(id, langCode, operatingSystem, rdx)
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(absManifestFilename); os.IsNotExist(err) {
		return nil, nil
	}

	manifestFile, err := os.Open(absManifestFilename)
	if err != nil {
		return nil, err
	}

	relFiles := make([]string, 0)
	manifestScanner := bufio.NewScanner(manifestFile)
	for manifestScanner.Scan() {
		relFiles = append(relFiles, manifestScanner.Text())
	}

	if err = manifestScanner.Err(); err != nil {
		return nil, err
	}

	return relFiles, nil
}

func appendManifest(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable, newRelFiles ...string) error {

	absManifestFilename, err := data.GetAbsManifestFilename(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	relFiles, err := readManifest(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	for _, nrf := range newRelFiles {
		if slices.Contains(relFiles, nrf) {
			continue
		}
		relFiles = append(relFiles, nrf)
	}

	absManifestDir, _ := filepath.Split(absManifestFilename)
	if _, err = os.Stat(absManifestDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absManifestDir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	manifestFile, err := os.Create(absManifestFilename)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	for _, relFile := range relFiles {
		if _, err = io.WriteString(manifestFile, relFile+"\n"); err != nil {
			return err
		}
	}

	return nil
}
