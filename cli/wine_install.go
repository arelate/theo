package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func WineInstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	_, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	var env []string
	if q.Has("env") {
		env = strings.Split(q.Get("env"), ",")
	}
	removeDownloads := !q.Has("keep-downloads")
	addSteamShortcut := !q.Has("no-steam-shortcut")
	verbose := q.Has("verbose")
	force := q.Has("force")

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	return WineInstall(langCode, env, downloadTypes, removeDownloads, addSteamShortcut, verbose, force, ids...)
}

func WineInstall(langCode string,
	env []string,
	downloadTypes []vangogh_integration.DownloadType,
	removeDownloads bool,
	addSteamShortcut bool,
	verbose bool,
	force bool,
	ids ...string) error {

	wia := nod.Begin("installing %s versions on %s...",
		vangogh_integration.Windows,
		data.CurrentOs())
	defer wia.EndWithResult("done")

	if data.CurrentOs() == vangogh_integration.Windows {
		wia.EndWithResult("WINE install is not required on Windows, use install")
		return nil
	}

	windowsOs := []vangogh_integration.OperatingSystem{vangogh_integration.Windows}
	langCodes := []string{langCode}

	notInstalled, err := wineFilterNotInstalled(langCode, ids...)
	if err != nil {
		return wia.EndWithError(err)
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
		return wia.EndWithError(err)
	}

	if err = Download(windowsOs, langCodes, downloadTypes, force, ids...); err != nil {
		return wia.EndWithError(err)
	}

	if err = Validate(windowsOs, langCodes, downloadTypes, ids...); err != nil {
		return wia.EndWithError(err)
	}

	if err = initPrefix(langCode, verbose, ids...); err != nil {
		return wia.EndWithError(err)
	}

	for _, id := range ids {
		if err := wineInstallProduct(id, langCode, env, downloadTypes, verbose, force); err != nil {
			return wia.EndWithError(err)
		}
	}

	if err := DefaultPrefixEnv(ids, langCode); err != nil {
		return wia.EndWithError(err)
	}

	if addSteamShortcut {
		if err := AddSteamShortcut(langCode, true, force, ids...); err != nil {
			return wia.EndWithError(err)
		}
	}

	if removeDownloads {
		if err = RemoveDownloads(windowsOs, langCodes, downloadTypes, force, ids...); err != nil {
			return wia.EndWithError(err)
		}
	}

	if err = pinInstalledMetadata(windowsOs, langCode, force, ids...); err != nil {
		return wia.EndWithError(err)
	}

	if err := RevealPrefix(langCode, ids...); err != nil {
		return wia.EndWithError(err)
	}

	return nil
}

func wineFilterNotInstalled(langCode string, ids ...string) ([]string, error) {

	notInstalled := make([]string, 0, len(ids))

	for _, id := range ids {

		absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
		if err != nil {
			return nil, err
		}

		absPrefixDriveCDir := filepath.Join(absPrefixDir, relPrefixDriveCDir)

		if _, err := os.Stat(absPrefixDriveCDir); err == nil {
			continue
		}

		notInstalled = append(notInstalled, id)
	}

	return notInstalled, nil
}

func wineInstallProduct(id, langCode string, env []string, downloadTypes []vangogh_integration.DownloadType, verbose, force bool) error {

	currentOs := data.CurrentOs()

	wipa := nod.Begin("installing %s version on %s...", vangogh_integration.Windows, currentOs)
	defer wipa.EndWithResult("done")

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return wipa.EndWithError(err)
	}

	metadata, err := getTheoMetadata(id, force)
	if err != nil {
		return wipa.EndWithError(err)
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(vangogh_integration.Windows).
		FilterLanguageCodes(langCode).
		FilterDownloadTypes(downloadTypes...)

	for _, link := range dls {
		linkExt := filepath.Ext(link.LocalFilename)
		if linkExt != exeExt {
			continue
		}
		absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

		switch currentOs {
		case vangogh_integration.MacOS:
			if err := macOsWineRun(id, langCode, env, verbose, absInstallerPath,
				"/VERYSILENT", "/NORESTART", "/CLOSEAPPLICATIONS"); err != nil {
				return nil
			}
		case vangogh_integration.Linux:
			if err := linuxWineRun(id, langCode, env, verbose, force, absInstallerPath,
				"/VERYSILENT", "/NORESTART", "/CLOSEAPPLICATIONS"); err != nil {
				return nil
			}
		default:
			panic("not implemented")
		}
	}

	return nil
}

func initPrefix(langCode string, verbose bool, ids ...string) error {

	cpa := nod.NewProgress("initializing prefixes for %s...", strings.Join(ids, ","))
	defer cpa.EndWithResult("done")

	cpa.TotalInt(len(ids))

	for _, id := range ids {

		switch data.CurrentOs() {
		case vangogh_integration.MacOS:
			if err := macOsInitPrefix(id, langCode, verbose); err != nil {
				return cpa.EndWithError(err)
			}
		case vangogh_integration.Linux:
			if err := linuxInitPrefix(id, langCode, verbose); err != nil {
				return cpa.EndWithError(err)
			}
		default:
			panic("not implemented")
		}

		cpa.Increment()
	}

	return nil
}
