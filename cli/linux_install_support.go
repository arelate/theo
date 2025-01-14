package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"os"
	"os/exec"
	"path/filepath"
)

const desktopGlob = "*.desktop"

func linuxExecuteInstaller(absInstallerPath, productInstalledAppDir string) error {

	_, fp := filepath.Split(absInstallerPath)

	leia := nod.Begin(" executing %s, please wait...", fp)
	defer leia.EndWithResult("done")

	// https://www.reddit.com/r/linux_gaming/comments/42l258/fully_automated_gog_games_install_howto/
	cmd := exec.Command(absInstallerPath, "--", "--i-agree-to-all-licenses", "--noreadme", "--nooptions", "--noprompt", "--destination", productInstalledAppDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func linuxPostDownloadActions(id string, link *vangogh_integration.TheoDownloadLink) error {

	lpda := nod.Begin(" performing %s post-download actions for %s...", vangogh_integration.Linux, id)
	defer lpda.EndWithResult("done")

	if data.CurrentOS() != vangogh_integration.Linux {
		return lpda.EndWithError(errors.New("Linux post-download actions are only supported on Linux"))
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return lpda.EndWithError(err)
	}

	productInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

	return chmodExecutable(productInstallerPath)
}

func chmodExecutable(path string) error {

	cea := nod.Begin(" setting executable attribute...")
	defer cea.EndWithResult("done")

	// chmod +x path/to/file
	cmd := exec.Command("chmod", "+x", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func snapshotDesktopFiles() ([]string, error) {

	desktopFiles := make([]string, 0)

	uhd, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	desktopDir := filepath.Join(uhd, "Desktop")

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return nil, err
	}

	applicationsDir := filepath.Join(udhd, "applications")

	for _, dir := range []string{desktopDir, applicationsDir} {

		globPath := filepath.Join(dir, desktopGlob)
		matches, err := filepath.Glob(globPath)
		if err != nil {
			return nil, err
		}

		desktopFiles = append(desktopFiles, matches...)
	}

	return desktopFiles, nil
}
