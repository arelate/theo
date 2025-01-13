package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"net/url"
)

func WineInstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	_, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	removeDownloads := !q.Has("keep-downloads")
	addSteamShortcut := !q.Has("no-steam-shortcut")
	force := q.Has("force")

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	return WineInstall(langCode, downloadTypes, removeDownloads, addSteamShortcut, force, ids...)
}

func WineInstall(langCode string,
	downloadTypes []vangogh_integration.DownloadType,
	removeDownloads bool,
	addSteamShortcut bool,
	force bool,
	ids ...string) error {

	// filter not installed (existing prefix)
	// backup metadata
	// download Windows versions
	// validate Windows versions
	// wineInstallProduct
	//	- run installer with this prefix as a target?
	// addWineSteamShortcut
	// removeDownloads
	// pinInstalledMetadata
	// reveal?

	return nil
}
