package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path/filepath"
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
	verbose := q.Has("verbose")

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	ip := &installParameters{
		operatingSystem: vangogh_integration.Windows,
		langCode:        langCode,
		downloadTypes:   downloadTypes,
		keepDownloads:   q.Has("keep-downloads"),
		noSteamShortcut: q.Has("no-steam-shortcut"),
		reveal:          q.Has("reveal"),
		force:           q.Has("force"),
	}

	return WineInstall(ip, env, verbose, ids...)
}

func WineInstall(ip *installParameters, env []string, verbose bool, ids ...string) error {

	start := time.Now().UTC().Unix()

	wia := nod.Begin("installing %s versions on %s...",
		vangogh_integration.Windows,
		data.CurrentOs())
	defer wia.Done()

	if data.CurrentOs() == vangogh_integration.Windows {
		wia.EndWithResult("WINE install is not required on Windows, use install")
		return nil
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	windowsOs := []vangogh_integration.OperatingSystem{vangogh_integration.Windows}
	langCodes := []string{ip.langCode}

	var flattened bool
	if ids, flattened, err = gameProductTypesFlatMap(rdx, ip.force, ids...); err != nil {
		return err
	} else if flattened {
		wia.EndWithResult("installing PACK included games")
		return WineInstall(ip, env, verbose, ids...)
	}

	notInstalled, err := filterNotInstalled(vangogh_integration.Windows, ip.langCode, ids...)
	if err != nil {
		return err
	}

	if len(notInstalled) > 0 {
		if !ip.force {
			ids = notInstalled
		}
	} else if !ip.force {
		wia.EndWithResult("all requested products are already installed")
		return nil
	}

	binariesDir, err := pathways.GetAbsRelDir(data.Binaries)
	if err != nil {
		return err
	}

	if empty, err := isDirEmpty(binariesDir); empty && err == nil {
		if err = SetupWine(false); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if err = BackupMetadata(); err != nil {
		return err
	}

	if err = Download(windowsOs, langCodes, ip.downloadTypes, nil, rdx, ip.force, ids...); err != nil {
		return err
	}

	if err = Validate(windowsOs, langCodes, ip.downloadTypes, nil, rdx, ids...); err != nil {
		return err
	}

	if err = initPrefix(ip.langCode, verbose, rdx, ids...); err != nil {
		return err
	}

	for _, id := range ids {
		if err = wineInstallProduct(id, ip.langCode, rdx, env, ip.downloadTypes, verbose, ip.force); err != nil {
			return err
		}

		if err = createPrefixInstalledFilesInventory(id, ip.langCode, rdx, start); err != nil {
			return err
		}
	}

	if err = DefaultPrefixEnv(ip.langCode, ids...); err != nil {
		return err
	}

	if !ip.noSteamShortcut {
		if err := AddSteamShortcut(SteamShortcutTargetWineRun, ip.langCode, rdx, ip.force, ids...); err != nil {
			return err
		}
	}

	if !ip.keepDownloads {
		if err = RemoveDownloads(windowsOs, langCodes, ip.downloadTypes, rdx, ip.force, ids...); err != nil {
			return err
		}
	}

	if err = pinInstalledDetails(windowsOs, ip.langCode, ip.force, ids...); err != nil {
		return err
	}

	if err = pinInstallParameters(ip, rdx, ids...); err != nil {
		return err
	}

	if err = setInstallDates(rdx, ids...); err != nil {
		return err
	}

	if ip.reveal {
		if err = RevealPrefix(ip.langCode, ids...); err != nil {
			return err
		}
	}

	return nil
}

func wineInstallProduct(id, langCode string, rdx redux.Writeable, env []string, downloadTypes []vangogh_integration.DownloadType, verbose, force bool) error {

	currentOs := data.CurrentOs()

	wipa := nod.Begin("installing %s version on %s...", vangogh_integration.Windows, currentOs)
	defer wipa.Done()

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	productDetails, err := GetProductDetails(id, rdx, force)
	if err != nil {
		return err
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	if err = hasFreeSpaceForProduct(productDetails, installedAppsDir,
		[]vangogh_integration.OperatingSystem{vangogh_integration.Windows}, []string{langCode}, downloadTypes, nil, force); err != nil {
		return err
	}

	dls := productDetails.DownloadLinks.
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

		et := &execTask{
			exe:     absInstallerPath,
			workDir: downloadsDir,
			args:    []string{innoSetupVerySilentArg, innoSetupNoRestartArg, innoSetupCloseApplicationsArg},
			env:     env,
			verbose: verbose,
		}

		if err = currentOsWineRun(id, langCode, rdx, et, force); err != nil {
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

func createPrefixInstalledFilesInventory(id, langCode string, rdx redux.Readable, utcTime int64) error {

	cpifma := nod.Begin(" creating installed files inventory...")
	defer cpifma.Done()

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	return createInventory(absPrefixDir, id, langCode, vangogh_integration.Windows, rdx, utcTime)
}
