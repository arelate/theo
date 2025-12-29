package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	desktopGlob  = "*.desktop"
	mojosetupDir = ".mojosetup"
)
const relLinuxGogGameInfoDir = "game"

const linuxStartShFilename = "start.sh"

const shExt = ".sh"

func linuxInstallProduct(id string,
	dls vangogh_integration.ProductDownloadLinks,
	rdx redux.Writeable) error {

	lia := nod.Begin("installing %s for %s...", id, vangogh_integration.Linux)
	defer lia.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	for _, link := range dls {

		if filepath.Ext(link.LocalFilename) != shExt {
			continue
		}

		absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

		if _, err := os.Stat(absInstallerPath); err != nil {
			return err
		}

		absInstalledPath, err := osInstalledPath(id, vangogh_integration.Linux, link.LanguageCode, rdx)
		if err != nil {
			return err
		}

		if err = linuxPostDownloadActions(id, &link); err != nil {
			return err
		}

		preInstallDesktopFiles, err := linuxSnapshotDesktopFiles()
		if err != nil {
			return err
		}

		if err = linuxExecuteInstaller(absInstallerPath, absInstalledPath); err != nil {
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

			if err = os.Remove(pidf); err != nil {
				return err
			}
		}

		mojosetupProductDir := filepath.Join(absInstalledPath, mojosetupDir)
		if _, err = os.Stat(mojosetupProductDir); err == nil {
			if err := os.RemoveAll(mojosetupProductDir); err != nil {
				return err
			}
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
		return errors.New("post-download Linux actions are only supported on Linux")
	}

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

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

func nixRunExecTask(et *execTask) error {

	nreta := nod.Begin(" running %s...", et.name)
	defer nreta.Done()

	cmd := exec.Command(et.exe, et.args...)
	cmd.Dir = et.workDir

	if et.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	for _, e := range et.env {
		cmd.Env = append(cmd.Env, e)
	}

	return cmd.Run()
}

func linuxFindStartSh(id, langCode string, rdx redux.Readable) (string, error) {

	absInstalledPath, err := osInstalledPath(id, vangogh_integration.Linux, langCode, rdx)
	if err != nil {
		return "", err
	}

	absStartShPath := filepath.Join(absInstalledPath, linuxStartShFilename)
	if _, err = os.Stat(absStartShPath); err == nil {
		return absStartShPath, nil
	} else if os.IsNotExist(err) {
		var matches []string
		if matches, err = filepath.Glob(filepath.Join(absInstalledPath, "*", linuxStartShFilename)); err == nil && len(matches) > 0 {
			return matches[0], nil
		}
	}

	return "", errors.New("cannot locate start.sh for " + id)
}

func nixUninstallProduct(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) error {

	umpa := nod.Begin(" uninstalling %s version of %s...", operatingSystem, id)
	defer umpa.Done()

	absBundlePath, err := osInstalledPath(id, operatingSystem, langCode, rdx)
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

func linuxFindGogGameInfo(id, langCode string, rdx redux.Readable) (string, error) {

	absInstalledPath, err := osInstalledPath(id, vangogh_integration.Linux, langCode, rdx)
	if err != nil {
		return "", err
	}

	gogGameInfoFilename := strings.Replace(gog_integration.GogGameInfoFilenameTemplate, "{id}", id, 1)

	absGogGameInfoPath := filepath.Join(absInstalledPath, relLinuxGogGameInfoDir, gogGameInfoFilename)

	if _, err = os.Stat(absGogGameInfoPath); err == nil {
		return absGogGameInfoPath, nil
	} else if os.IsNotExist(err) {
		return "", nil
	} else {
		return "", err
	}
}

func linuxExecTaskGogGameInfo(absGogGameInfoPath string, gogGameInfo *gog_integration.GogGameInfo, et *execTask) (*execTask, error) {

	pt, err := gogGameInfo.GetPlayTask(et.playTask)
	if err != nil {
		return nil, err
	}

	absGogGameInfoDir, _ := filepath.Split(absGogGameInfoPath)

	exePath := pt.Path
	// account for Windows-style relative paths, e.g. DOSBOX\DOSBOX.exe
	if parts := strings.Split(exePath, "\\"); len(parts) > 1 {
		exePath = filepath.Join(parts...)
	}

	absExePath := filepath.Join(absGogGameInfoDir, exePath)

	et.name = pt.Name
	et.exe = absExePath
	et.workDir = filepath.Join(absGogGameInfoDir, pt.WorkingDir)

	if pt.Arguments != "" {
		et.args = append(et.args, pt.Arguments)
	}

	return et, nil
}

func linuxExecTaskStartSh(absStartShPath string, et *execTask) (*execTask, error) {

	et.exe = absStartShPath

	return et, nil
}
