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
	removeDownloads := !q.Has("keep-downloads")
	addSteamShortcut := !q.Has("no-steam-shortcut")
	verbose := q.Has("verbose")
	force := q.Has("force")

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	return WineInstall(langCode, downloadTypes, removeDownloads, addSteamShortcut, verbose, force, ids...)
}

func WineInstall(langCode string,
	downloadTypes []vangogh_integration.DownloadType,
	removeDownloads bool,
	addSteamShortcut bool,
	verbose bool,
	force bool,
	ids ...string) error {

	wia := nod.Begin("installing %s versions on %s...",
		vangogh_integration.Windows,
		data.CurrentOS())
	defer wia.EndWithResult("done")

	if data.CurrentOS() == vangogh_integration.Windows {
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
		if err := wineInstallProduct(id, langCode, downloadTypes, verbose, force); err != nil {
			return wia.EndWithError(err)
		}
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

		if _, err := os.Stat(absPrefixDir); err == nil {
			continue
		}

		notInstalled = append(notInstalled, id)
	}

	return notInstalled, nil
}

func wineInstallProduct(id, langCode string, downloadTypes []vangogh_integration.DownloadType, verbose, force bool) error {

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

	for _, link := range dls {
		linkExt := filepath.Ext(link.LocalFilename)
		if linkExt != exeExt {
			continue
		}
		absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

		switch data.CurrentOS() {
		case vangogh_integration.MacOS:
			if err := macOsWineRun(id, langCode, nil, verbose, absInstallerPath, "/VERYSILENT", "/NORESTART", "/CLOSEAPPLICATIONS"); err != nil {
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

		switch data.CurrentOS() {
		case vangogh_integration.MacOS:
			if err := macOsInitPrefix(id, langCode, verbose); err != nil {
				return cpa.EndWithError(err)
			}
		case vangogh_integration.Linux:
			// do nothing, umu-launch will create prefix during installation
		default:
			panic("not implemented")
		}

		cpa.Increment()
	}

	return nil
}
