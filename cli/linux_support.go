package cli

import (
	"bytes"
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const desktopGlob = "*.desktop"

const mojosetupDir = ".mojosetup"

func linuxInstallProduct(id string,
	productDetails *vangogh_integration.ProductDetails,
	link *vangogh_integration.ProductDownloadLink,
	rdx redux.Writeable) error {

	lia := nod.Begin("installing %s version of %s...", vangogh_integration.Linux, productDetails.Title)
	defer lia.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
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

func linuxPostDownloadActions(id string, link *vangogh_integration.ProductDownloadLink) error {

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

func linuxExecute(path string, et *execTask) error {

	startShPath := linuxLocateStartSh(path)

	cmd := exec.Command(startShPath)
	cmd.Dir = path

	if et.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	for _, e := range et.env {
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

	if err = os.RemoveAll(absBundlePath); err != nil {
		return err
	}

	return nil
}

func nixFreeSpace(path string) (int64, error) {

	dfPath, err := exec.LookPath("df")
	if err != nil {
		return -1, err
	}

	buf := bytes.NewBuffer(nil)

	dfCmd := exec.Command(dfPath, "-k", path)
	dfCmd.Stdout = buf

	if err = dfCmd.Run(); err != nil {
		return -1, err
	}

	var lines []string
	if lines = strings.Split(buf.String(), "\n"); len(lines) < 2 {
		return -1, errors.New("unsupported df output lines format")
	}

	var ai int
	if ai = strings.Index(lines[0], "Available"); ai == 0 || ai >= len(lines[0])-1 {
		return -1, errors.New("df output is missing Available")
	}

	var sub string
	if sub = lines[1][ai:]; len(sub) == 0 {
		return -1, errors.New("df values format is too short")
	}

	var abs string
	var ok bool
	if abs, _, ok = strings.Cut(sub, " "); !ok {
		abs = sub
	}

	if abi, err := strconv.ParseInt(abs, 10, 32); err == nil {
		return abi * 1024, nil
	} else {
		return -1, err
	}
}
