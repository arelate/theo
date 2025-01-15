package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
)

func WineInstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	_, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	wineRepo := q.Get("wine-repo")
	removeDownloads := !q.Has("keep-downloads")
	addSteamShortcut := !q.Has("no-steam-shortcut")
	force := q.Has("force")

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	return WineInstall(langCode, wineRepo, downloadTypes, removeDownloads, addSteamShortcut, force, ids...)
}

func WineInstall(langCode string,
	wineRepo string,
	downloadTypes []vangogh_integration.DownloadType,
	removeDownloads bool,
	addSteamShortcut bool,
	force bool,
	ids ...string) error {

	wia := nod.Begin("installing products with WINE...")
	defer wia.EndWithResult("done")

	if data.CurrentOS() == vangogh_integration.Windows {
		wia.EndWithResult("WINE install is not supported on Windows")
		return nil
	}

	windowsOs := []vangogh_integration.OperatingSystem{vangogh_integration.Windows}
	langCodes := []string{langCode}

	vangogh_integration.PrintParams(ids, windowsOs, langCodes, downloadTypes, true)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return wia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SlugProperty)
	if err != nil {
		return wia.EndWithError(err)
	}

	notInstalled, err := wineFilterNotInstalled(langCode, rdx, ids...)
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

	if err = initPrefix(langCode, wineRepo, force, ids...); err != nil {
		return wia.EndWithError(err)
	}

	for _, id := range ids {
		if err := wineInstallProduct(id, langCode, downloadTypes, force); err != nil {
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

	if err = pinInstalledMetadata(windowsOs, force, ids...); err != nil {
		return wia.EndWithError(err)
	}

	if err := RevealPrefix(langCode, ids...); err != nil {
		return wia.EndWithError(err)
	}

	return nil
}

func wineFilterNotInstalled(langCode string, rdx kevlar.ReadableRedux, ids ...string) ([]string, error) {

	notInstalled := make([]string, 0, len(ids))

	for _, id := range ids {

		ok, err := productPrefixExists(id, langCode, rdx)
		if err != nil {
			return nil, err
		}

		if ok {
			continue
		}

		notInstalled = append(notInstalled, id)
	}

	return notInstalled, nil
}

func productPrefixExists(id, langCode string, rdx kevlar.ReadableRedux) (bool, error) {

	prefixName, err := data.GetPrefixName(id, langCode, rdx)
	if err != nil {
		return false, err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(prefixName)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(absPrefixDir); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func wineInstallProduct(id, langCode string, downloadTypes []vangogh_integration.DownloadType, force bool) error {
	return nil
}
