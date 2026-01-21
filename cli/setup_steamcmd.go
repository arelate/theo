package cli

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

var steamCmdBinary = map[vangogh_integration.OperatingSystem]string{
	vangogh_integration.MacOS: "steamcmd.sh",
	vangogh_integration.Linux: "steamcmd.sh",
}

func SetupSteamCmdHandler(u *url.URL) error {

	q := u.Query()

	verbose := q.Has("verbose")
	force := q.Has("force")

	return SetupSteamCmd(verbose, force)
}

func SetupSteamCmd(verbose, force bool) error {

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

	if err = updateSteamCmdRuntime(currentOs, verbose); err != nil {
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

	absSteamCmdBinaryPath, err := steamCmdBinaryPath(operatingSystem)
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

func updateSteamCmdRuntime(operatingSystem vangogh_integration.OperatingSystem, verbose bool) error {

	uscra := nod.Begin(" updating SteamCMD runtime for %s, please wait...", operatingSystem)
	defer uscra.Done()

	absSteamCmdBinaryPath, err := steamCmdBinaryPath(operatingSystem)
	if err != nil {
		return err
	}

	cmd := exec.Command(absSteamCmdBinaryPath, "+quit")

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

func steamCmdBinaryPath(operatingSystem vangogh_integration.OperatingSystem) (string, error) {
	switch operatingSystem {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		steamCmdBinariesDir := data.Pwd.AbsRelDirPath(data.BinUnpacks, data.SteamCmd)
		osSteamCmdBinariesDir := filepath.Join(steamCmdBinariesDir, operatingSystem.String())
		return filepath.Join(osSteamCmdBinariesDir, steamCmdBinary[operatingSystem]), nil
	default:
		return "", operatingSystem.ErrUnsupported()
	}
}
