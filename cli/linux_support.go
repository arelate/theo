package cli

import (
	"errors"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const desktopGlob = "*.desktop"

const mojosetupDir = ".mojosetup"

func linuxInstallProduct(id string,
	metadata *vangogh_integration.TheoMetadata,
	link *vangogh_integration.TheoDownloadLink,
	absInstallerPath, installedAppsDir string,
	rdx kevlar.WriteableRedux) error {

	lia := nod.Begin("installing %s version of %s...", vangogh_integration.Linux, metadata.Title)
	defer lia.EndWithResult("done")

	if err := rdx.MustHave(data.SlugProperty, data.BundleNameProperty); err != nil {
		return lia.EndWithError(err)
	}

	if _, err := os.Stat(absInstallerPath); err != nil {
		return lia.EndWithError(err)
	}

	productTitle, _ := rdx.GetLastVal(data.SlugProperty, id)

	if err := rdx.ReplaceValues(data.BundleNameProperty, id, productTitle); err != nil {
		return lia.EndWithError(err)
	}

	osLangCodeDir := data.OsLangCodeDir(vangogh_integration.Linux, link.LanguageCode)
	productInstalledAppDir := filepath.Join(installedAppsDir, osLangCodeDir, productTitle)

	if err := linuxPostDownloadActions(id, link); err != nil {
		return lia.EndWithError(err)
	}

	preInstallDesktopFiles, err := linuxSnapshotDesktopFiles()
	if err != nil {
		return lia.EndWithError(err)
	}

	fmt.Println(preInstallDesktopFiles)

	if err := linuxExecuteInstaller(absInstallerPath, productInstalledAppDir); err != nil {
		return lia.EndWithError(err)
	}

	postInstallDesktopFiles, err := linuxSnapshotDesktopFiles()
	if err != nil {
		return lia.EndWithError(err)
	}

	for _, pidf := range postInstallDesktopFiles {
		if slices.Contains(preInstallDesktopFiles, pidf) {
			continue
		}

		if err := os.Remove(pidf); err != nil {
			return lia.EndWithError(err)
		}
	}

	mojosetupProductDir := filepath.Join(productInstalledAppDir, mojosetupDir)
	if _, err = os.Stat(mojosetupProductDir); err == nil {
		if err := os.RemoveAll(mojosetupProductDir); err != nil {
			return lia.EndWithError(err)
		}
	}

	fmt.Println(postInstallDesktopFiles)

	return nil
}

func linuxExecuteInstaller(absInstallerPath, productInstalledAppDir string) error {

	_, fp := filepath.Split(absInstallerPath)

	leia := nod.Begin(" executing %s, please wait...", fp)
	defer leia.EndWithResult("done")

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

func linuxPostDownloadActions(id string, link *vangogh_integration.TheoDownloadLink) error {

	lpda := nod.Begin(" performing %s post-download actions for %s...", vangogh_integration.Linux, id)
	defer lpda.EndWithResult("done")

	if data.CurrentOs() != vangogh_integration.Linux {
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

func nixUninstallProduct(title string, operatingSystem vangogh_integration.OperatingSystem, installationDir, langCode, bundleName string) error {

	umpa := nod.Begin(" uninstalling %s version of %s...", operatingSystem, title)
	defer umpa.EndWithResult("done")

	if bundleName == "" {
		umpa.EndWithResult("product must have bundle name for uninstall")
		return nil
	}

	osLangCodeDir := data.OsLangCodeDir(operatingSystem, langCode)
	bundlePath := filepath.Join(installationDir, osLangCodeDir, bundleName)

	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		umpa.EndWithResult("not present")
		return nil
	}

	if err := os.RemoveAll(bundlePath); err != nil {
		return err
	}

	return nil
}
