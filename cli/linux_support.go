package cli

import (
	"errors"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const desktopGlob = "*.desktop"

const mojosetupDir = ".mojosetup"

func linuxInstallProduct(id string,
	downloadsManifest *vangogh_integration.DownloadsManifest,
	link *vangogh_integration.ManifestDownloadLink,
	rdx redux.Writeable) error {

	lia := nod.Begin("installing %s version of %s...", vangogh_integration.Linux, downloadsManifest.Title)
	defer lia.Done()

	if err := rdx.MustHave(data.SlugProperty, data.BundleNameProperty); err != nil {
		return err
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

	if _, err = os.Stat(absInstallerPath); err != nil {
		return err
	}

	absBundlePath, err := data.GetAbsBundlePath(id, link.LanguageCode, vangogh_integration.Linux, rdx)
	if err != nil {
		return err
	}

	if err := linuxPostDownloadActions(id, link); err != nil {
		return err
	}

	preInstallDesktopFiles, err := linuxSnapshotDesktopFiles()
	if err != nil {
		return err
	}

	fmt.Println(preInstallDesktopFiles)

	if err := linuxExecuteInstaller(absInstallerPath, absBundlePath); err != nil {
		return err
	}

	postInstallDesktopFiles, err := linuxSnapshotDesktopFiles()
	if err != nil {
		return err
	}

	for _, pidf := range postInstallDesktopFiles {
		if slices.Contains(preInstallDesktopFiles, pidf) {
			continue
		}

		if err := os.Remove(pidf); err != nil {
			return err
		}
	}

	mojosetupProductDir := filepath.Join(absBundlePath, mojosetupDir)
	if _, err = os.Stat(mojosetupProductDir); err == nil {
		if err := os.RemoveAll(mojosetupProductDir); err != nil {
			return err
		}
	}

	fmt.Println(postInstallDesktopFiles)

	return nil
}

func linuxExecuteInstaller(absInstallerPath, productInstalledAppDir string) error {

	_, fp := filepath.Split(absInstallerPath)

	leia := nod.Begin(" executing %s, please wait...", fp)
	defer leia.Done()

	// https://www.reddit.com/r/linux_gaming/comments/42l258/fully_automated_gog_games_install_howto/
	// tl;dr; those flags are required, but not sufficient. Installing installer and then DLC will
	// normally trigger additional prompts. Details:
	// Note how linuxSnapshotDesktopFiles is used pre- and post- install to remove
	// .desktop files created by the installer. This is notable because if those files are not
	// removed and DLCs are installed they will attempt to create the same files and will ask
	// to confirm to overwrite, interrupting automated installation.
	cmd := exec.Command(absInstallerPath, "--", "--i-agree-to-all-licenses", "--noreadme", "--nooptions", "--noprompt", "--destination", productInstalledAppDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func linuxPostDownloadActions(id string, link *vangogh_integration.ManifestDownloadLink) error {

	lpda := nod.Begin(" performing %s post-download actions for %s...", vangogh_integration.Linux, id)
	defer lpda.Done()

	if data.CurrentOs() != vangogh_integration.Linux {
		return errors.New("Linux post-download actions are only supported on Linux")
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	productInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

	return chmodExecutable(productInstallerPath)
}

func chmodExecutable(path string) error {

	cea := nod.Begin(" setting executable attribute...")
	defer cea.Done()

	// chmod +x path/to/file
	cmd := exec.Command("chmod", "+x", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func linuxSnapshotDesktopFiles() ([]string, error) {

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

func linuxReveal(path string) error {
	cmd := exec.Command("xdg-open", path)
	return cmd.Run()
}

func linuxExecute(path string, env []string, verbose bool) error {

	startShPath := linuxLocateStartSh(path)

	cmd := exec.Command(startShPath)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	for _, e := range env {
		cmd.Env = append(cmd.Env, e)
	}

	return cmd.Run()
}

func linuxLocateStartSh(path string) string {
	if strings.HasSuffix(path, linuxStartShFilename) {
		return path
	}

	absStartShPath := filepath.Join(path, linuxStartShFilename)
	if _, err := os.Stat(absStartShPath); err == nil {
		return absStartShPath
	} else if os.IsNotExist(err) {
		if matches, err := filepath.Glob(filepath.Join(path, "*", linuxStartShFilename)); err == nil && len(matches) > 0 {
			return matches[0]
		}
	}

	return path
}

func nixUninstallProduct(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) error {

	umpa := nod.Begin(" uninstalling %s version of %s...", operatingSystem, id)
	defer umpa.Done()

	absBundlePath, err := data.GetAbsBundlePath(id, langCode, operatingSystem, rdx)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absBundlePath); os.IsNotExist(err) {
		umpa.EndWithResult("not present")
		return nil
	}

	// TODO: similarly to wine-uninstall - use manifests to remove individual files
	if err := os.RemoveAll(absBundlePath); err != nil {
		return err
	}

	return nil
}
