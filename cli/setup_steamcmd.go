package cli

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func SetupSteamCmdHandler(u *url.URL) error {

	q := u.Query()

	force := q.Has("force")

	return SetupSteamCmd(force)
}

func SetupSteamCmd(force bool) error {

	currentOs := data.CurrentOs()

	ssca := nod.Begin("setting up SteamCMD for %s...", currentOs)
	defer ssca.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.VangoghProperties()...)
	if err != nil {
		return err
	}

	if err = vangoghValidateSessionToken(rdx); err != nil {
		return err
	}

	if err = downloadSteamCmdBinaries(currentOs, rdx, force); err != nil {
		return err
	}

	if err = unpackSteamCmdBinaries(currentOs, force); err != nil {
		return err
	}

	return nil
}

func downloadSteamCmdBinaries(operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable, force bool) error {

	dscba := nod.NewProgress(" downloading SteamCMD for %s...", operatingSystem)
	defer dscba.Done()

	steamCmdDownloads := data.Pwd.AbsRelDirPath(data.BinDownloads, data.SteamCmd)

	query := url.Values{
		vangogh_integration.OperatingSystemsProperty: {operatingSystem.String()},
	}

	steamCmdBinaryUrl, err := data.VangoghUrl(data.HttpSteamCmdBinaryFilePath, query, rdx)
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if token, ok := rdx.GetLastVal(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty); ok && token != "" {
		dc.SetAuthorizationBearer(token)
	}

	relSteamCmdFilename := filepath.Base(steamcmd.Urls[operatingSystem])

	return dc.Download(steamCmdBinaryUrl, force, dscba, steamCmdDownloads, relSteamCmdFilename)
}

func unpackSteamCmdBinaries(operatingSystem vangogh_integration.OperatingSystem, force bool) error {

	uscba := nod.Begin(" unpacking SteamCMD for %s...", operatingSystem)
	defer uscba.Done()

	absSteamCmdBinaryPath, err := data.AbsSteamCmdBinPath(operatingSystem)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absSteamCmdBinaryPath); err == nil && !force {
		uscba.EndWithResult("already unpacked")
		return nil
	}

	steamCmdDownloadsDir := data.Pwd.AbsRelDirPath(data.BinDownloads, data.SteamCmd)
	absSteamCmdDownload := filepath.Join(steamCmdDownloadsDir, filepath.Base(steamcmd.Urls[operatingSystem]))

	steamCmdBinariesDir := data.Pwd.AbsRelDirPath(data.BinUnpacks, data.SteamCmd)
	osSteamCmdBinariesDir := filepath.Join(steamCmdBinariesDir, operatingSystem.String())

	return untar(absSteamCmdDownload, osSteamCmdBinariesDir)
}
