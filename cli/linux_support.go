package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	relLinuxGogGameInfoDir = "game"
	linuxStartShFilename   = "start.sh"
	shExt                  = ".sh"
	relExtractedDataPath   = "data/noarch"
)

func linuxUnpackInstallers(id string, dls vangogh_integration.ProductDownloadLinks, unpackDir string) error {

	luida := nod.Begin("unpacking %s installers for %s...", vangogh_integration.Linux, id)
	defer luida.Done()

	for _, link := range dls {

		if !isLinkExecutable(&link, vangogh_integration.Linux) {
			continue
		}

		downloadsDir := data.Pwd.AbsDirPath(data.Downloads)
		linkInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

		absUnpackDir := filepath.Join(unpackDir, link.LocalFilename)

		if err := linuxExtractInstallerData(linkInstallerPath, absUnpackDir); err != nil {
			return err
		}
	}

	return nil
}

func linuxExtractInstallerData(linkInstallerPath, absUnpackDir string) error {

	_, filename := filepath.Split(linkInstallerPath)

	leida := nod.Begin(" extracting %s data...", filename)
	defer leida.Done()

	if err := MojosetupExtract(new(ExtractOptions{Src: linkInstallerPath, Dst: absUnpackDir, Data: true})); err != nil {
		return err
	}

	absDataPath := filepath.Join(absUnpackDir, DataFn)

	return untar(absDataPath, absUnpackDir)
}

func linuxPlaceUnpackedFiles(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Readable, unpackDir string) error {

	lufa := nod.Begin(" placing unpacked files for %s...", id)
	defer lufa.Done()

	for _, link := range dls {

		if !isLinkExecutable(&link, vangogh_integration.Linux) {
			continue
		}

		absUnpackedPath := filepath.Join(unpackDir, link.LocalFilename, relExtractedDataPath)
		if _, err := os.Stat(absUnpackedPath); os.IsNotExist(err) {
			return ErrMissingExtractedPayload
		}

		installedAppPath, err := originOsInstalledPath(id, ii, rdx)

		if err = vangoghPlaceUnpackedLinkPayload(&link, absUnpackedPath, installedAppPath); err != nil {
			return err
		}
	}

	return nil
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

func linuxReveal(path string) error {
	cmd := exec.Command("xdg-open", path)
	return cmd.Run()
}

func nixRunExecTask(et *execTask) error {

	nreta := nod.Begin(" running %s...", et.title)
	defer nreta.Done()

	if data.CurrentOs() == vangogh_integration.MacOS &&
		strings.HasSuffix(et.exe, appBundleExt) {
		et.args = append([]string{et.exe, "--args"}, et.args...)
		et.exe = "open"
	}

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

func linuxFindStartSh(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {

	absInstalledPath, err := originOsInstalledPath(id, ii, rdx)
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

func nixFreeSpace(path string) (int64, error) {

	dfPath, err := exec.LookPath("df")
	if err != nil {
		return -1, err
	}

	buf := bytes.NewBuffer(nil)
	dfCmd := exec.Command(dfPath, "-Pk", path)
	dfCmd.Stdout = buf

	if err = dfCmd.Run(); err != nil {
		return -1, err
	}

	var lines []string
	if lines = strings.Split(buf.String(), "\n"); len(lines) < 2 {
		return -1, errors.New("unsupported df output lines format")
	}

	values := make([]string, 0, 5)
	for _, val := range strings.Split(lines[1], " ") {
		if val == "" {
			continue
		}
		values = append(values, val)
	}

	if len(values) < 4 {
		return -1, errors.New("unsupported df output values format")
	}

	var abi int64
	// When both the -k and -P options are specified, the following header line shall be written (in the POSIX locale):
	//"Filesystem 1024-blocks Used Available Capacity Mounted on\n"
	if abi, err = strconv.ParseInt(values[3], 10, 64); err == nil {
		return abi * 1024, nil
	} else {
		return -1, err
	}
}

func linuxFindGogGameInfo(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {

	absInstalledPath, err := originOsInstalledPath(id, ii, rdx)
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

	pt, err := gogGameInfo.GetPlayTask(et.task)
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

	et.title = pt.Name
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
