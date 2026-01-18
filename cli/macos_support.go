package cli

import (
	"errors"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	catCmdPfx    = "cat "
	appBundleExt = ".app"
)

const (
	installerTypeGame = "game"
	installerTypeDlc  = "dlc"
)

const relMacOsGogGameInfoDir = "Contents/Resources"

const pkgExt = ".pkg"

const (
	relPayloadPath = "package.pkg/Scripts/payload"
)

//func removeExistingCreateMissing(path string, force bool) error {
//	if _, err := os.Stat(path); err == nil {
//		if force {
//			if err = os.RemoveAll(path); err != nil {
//				return err
//			}
//		} else {
//			return nil
//		}
//	}
//
//	pathDir, _ := filepath.Split(path)
//	if _, err := os.Stat(pathDir); os.IsNotExist(err) {
//		if err = os.MkdirAll(pathDir, 0755); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}

func macOsUnpackInstallers(id string, dls vangogh_integration.ProductDownloadLinks, dstDir string, force bool) error {

	mui := nod.Begin(" unpacking %s installers with pkgutil, please wait...", id)
	defer mui.Done()

	downloadsDir := data.Pwd.AbsDirPath(vangogh_integration.Downloads)
	productDownloadsDir := filepath.Join(downloadsDir, id)

	for _, link := range dls {

		if filepath.Ext(link.LocalFilename) != pkgExt {
			continue
		}

		absInstallerPath := filepath.Join(productDownloadsDir, link.LocalFilename)

		if err := macOsPkgUtilExtractAll(&link, absInstallerPath, dstDir, force); err != nil {
			return err
		}

	}

	return nil
}

func macOsPkgUtilExtractAll(link *vangogh_integration.ProductDownloadLink, srcPath, dstPath string, force bool) error {

	mpuea := nod.Begin(" unpacking %s with pkgutil, please wait...", link.LocalFilename)
	defer mpuea.Done()

	localFilenameDstDir := filepath.Join(dstPath, link.LocalFilename)

	pathDir, _ := filepath.Split(localFilenameDstDir)
	if _, err := os.Stat(pathDir); os.IsNotExist(err) {
		if err = os.MkdirAll(pathDir, 0755); err != nil {
			return err
		}
	}

	cmd := exec.Command("pkgutil", "--expand-full", srcPath, localFilenameDstDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func macOsPlaceUnpackedFiles(id string, dls vangogh_integration.ProductDownloadLinks, rdx redux.Readable, unpackDir string) error {

	mpufa := nod.Begin(" placing product installation files for %s...", id)
	defer mpufa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	for _, link := range dls {

		if filepath.Ext(link.LocalFilename) != pkgExt {
			continue
		}

		absPostInstallScriptPath := PostInstallScriptPath(unpackDir, &link)
		postInstallScript, err := ParsePostInstallScript(absPostInstallScriptPath)
		if err != nil {
			return err
		}

		absUnpackedPayloadPath := filepath.Join(unpackDir, link.LocalFilename, relPayloadPath)
		if _, err = os.Stat(absUnpackedPayloadPath); os.IsNotExist(err) {
			return errors.New("cannot locate extracted payload")
		}

		installerType := postInstallScript.InstallerType()

		absBundlePath, err := osInstalledPath(id, vangogh_integration.MacOS, link.LanguageCode, rdx)
		if strings.HasSuffix(postInstallScript.bundleName, appBundleExt) {
			absBundlePath = filepath.Join(absBundlePath, postInstallScript.bundleName)
		}

		switch installerType {
		case installerTypeGame:
			fallthrough
		case installerTypeDlc:
			if err = macOsPlaceUnpackedPayload(&link, absUnpackedPayloadPath, absBundlePath); err != nil {
				return err
			}
		default:
			return errors.New("unknown postinstall script installer type: " + installerType)
		}

	}

	return nil
}

func macOsPlaceUnpackedPayload(link *vangogh_integration.ProductDownloadLink, absUnpackedPayloadPath, absInstallationPath string) error {

	mpda := nod.Begin(" placing unpacked files from %s...", link.LocalFilename)
	defer mpda.Done()

	if _, err := os.Stat(absInstallationPath); os.IsNotExist(err) {
		if err = os.MkdirAll(absInstallationPath, 0755); err != nil {
			return err
		}
	}

	// enumerate all DLC files in the payload directory
	relDlcFiles, err := relWalkDir(absUnpackedPayloadPath)
	if err != nil {
		return err
	}

	for _, dlcFile := range relDlcFiles {

		absDstPath := filepath.Join(absInstallationPath, dlcFile)
		absDstDir, _ := filepath.Split(absDstPath)

		if _, err = os.Stat(absDstDir); os.IsNotExist(err) {
			if err = os.MkdirAll(absDstDir, 0755); err != nil {
				return err
			}
		}

		absSrcPath := filepath.Join(absUnpackedPayloadPath, dlcFile)

		if err = os.Rename(absSrcPath, absDstPath); err != nil {
			return err
		}
	}

	return nil
}

func macOsGetInventory(dls vangogh_integration.ProductDownloadLinks, unpackDir string) ([]string, error) {

	mgia := nod.Begin(" creating inventory of unpacked files...")
	defer mgia.Done()

	filesMap := make(map[string]any)

	for _, link := range dls {

		if filepath.Ext(link.LocalFilename) != pkgExt {
			continue
		}

		absUnpackedPayloadPath := filepath.Join(unpackDir, link.LocalFilename, relPayloadPath)
		if _, err := os.Stat(absUnpackedPayloadPath); os.IsNotExist(err) {
			return nil, errors.New("cannot locate extracted payload")
		}

		relUnpackedFiles, err := relWalkDir(absUnpackedPayloadPath)
		if err != nil {
			return nil, err
		}

		absPostInstallScriptPath := PostInstallScriptPath(unpackDir, &link)
		postInstallScript, err := ParsePostInstallScript(absPostInstallScriptPath)
		if err != nil {
			return nil, err
		}

		var appBundleName string
		if strings.HasSuffix(postInstallScript.bundleName, appBundleExt) {
			appBundleName = postInstallScript.bundleName
		}

		for _, ruf := range relUnpackedFiles {
			if appBundleName != "" {
				ruf = filepath.Join(appBundleName, ruf)
			}
			filesMap[ruf] = nil
		}

	}

	return slices.Sorted(maps.Keys(filesMap)), nil
}

func macOsPostInstallActions(id string,
	dls vangogh_integration.ProductDownloadLinks,
	rdx redux.Readable,
	unpackDir string) error {

	mpia := nod.Begin(" performing post-install %s actions for %s...", vangogh_integration.MacOS, id)
	defer mpia.Done()

	absBundlePaths := make(map[string]any)

	for _, link := range dls {

		if filepath.Ext(link.LocalFilename) != pkgExt {
			continue
		}

		if filepath.Ext(link.LocalFilename) != pkgExt {
			// for macOS - there's nothing to be done for additional files (that are not .pkg installers)
			return nil
		}

		downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

		productDownloadsDir := filepath.Join(downloadsDir, id)

		absPostInstallScriptPath := PostInstallScriptPath(unpackDir, &link)

		pis, err := ParsePostInstallScript(absPostInstallScriptPath)
		if err != nil {
			return err
		}

		absBundlePath, err := osInstalledPath(id, vangogh_integration.MacOS, link.LanguageCode, rdx)
		if err != nil {
			return err
		}

		absBundlePaths[absBundlePath] = nil

		if customCommands := pis.CustomCommands(); len(customCommands) > 0 {
			if err = macOsProcessPostInstallScript(customCommands, productDownloadsDir, absBundlePath); err != nil {
				return err
			}
		}
	}

	for absBundlePath := range absBundlePaths {
		if err := macOsRemoveXattrs(absBundlePath); err != nil {
			return err
		}
	}

	return nil
}

func macOsRemoveXattrs(path string) error {

	mrxa := nod.Begin(" removing xattrs...")
	defer mrxa.Done()

	// xattr -cr /Applications/Bundle Name.app
	cmd := exec.Command("xattr", "-cr", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func macOsProcessPostInstallScript(commands []string, productDownloadsDir, bundleAppPath string) error {

	pcca := nod.NewProgress(" processing post-install commands...")
	defer pcca.Done()

	pcca.TotalInt(len(commands))

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, catCmdPfx) {
			if catCmdParts := strings.Split(strings.TrimPrefix(cmd, catCmdPfx), " "); len(catCmdParts) == 3 {
				srcGlob := strings.Trim(strings.Replace(catCmdParts[0], "\"${pkgpath}\"", productDownloadsDir, 1), "\"")
				dstPath := strings.Trim(strings.Replace(catCmdParts[2], "${gog_full_path}", bundleAppPath, 1), "\"")
				if err := macOsCatFiles(srcGlob, dstPath); err != nil {
					return err
				}
			}
			pcca.Increment()
			continue
		}
		// at this point we've handled all known commands, so anything here would be unknown
		return errors.New("cannot process unknown custom command: " + cmd)
	}
	return nil
}

func macOsCatFiles(srcGlob string, dstPath string) error {

	if srcGlob == "" {
		return errors.New("cat command source glob cannot be empty")
	}
	if dstPath == "" {
		return errors.New("cat command destination path cannot be empty")
	}

	if matches, err := filepath.Glob(srcGlob); err == nil && len(matches) == 0 {
		return errors.New("no files match pattern: " + srcGlob)
	}

	_, srcFileGlob := filepath.Split(srcGlob)
	_, dstFilename := filepath.Split(dstPath)

	ecfa := nod.NewProgress(" cat downloads...%s into installed-apps...%s", srcFileGlob, dstFilename)
	defer ecfa.Done()

	dstDir, _ := filepath.Split(dstPath)
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		if err = os.MkdirAll(dstDir, 0755); err != nil {
			return err
		}
	}

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	matches, err := filepath.Glob(srcGlob)
	if err != nil {
		return err
	}

	slices.Sort(matches)

	ecfa.TotalInt(len(matches))

	for _, match := range matches {

		srcFile, err := os.Open(match)
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			_ = srcFile.Close()
			return err
		}

		ecfa.Increment()
		_ = srcFile.Close()

		if err := os.Remove(match); err != nil {
			return err
		}
	}

	return nil
}

func macOsReveal(path string) error {
	cmd := exec.Command("open", "-R", path)
	return cmd.Run()
}

func macOsFindGogGameInfo(id, langCode string, rdx redux.Readable) (string, error) {

	absBundlePath, err := osInstalledPath(id, vangogh_integration.MacOS, langCode, rdx)
	if err != nil {
		return "", err
	}

	gogGameInfoFilename := strings.Replace(gog_integration.GogGameInfoFilenameTemplate, "{id}", id, 1)

	absGogGameInfoPath := filepath.Join(absBundlePath, relMacOsGogGameInfoDir, gogGameInfoFilename)

	if _, err = os.Stat(absGogGameInfoPath); err == nil {
		return absGogGameInfoPath, nil
	} else if os.IsNotExist(err) {
		// some GOG games put Contents/Resources in the top install location, not app bundle

		var absInstalledPath string
		absInstalledPath, err = osInstalledPath(id, vangogh_integration.MacOS, langCode, rdx)
		if err != nil {
			return "", err
		}

		absGogGameInfoPath = filepath.Join(absInstalledPath, relMacOsGogGameInfoDir, gogGameInfoFilename)
		if _, err = os.Stat(absGogGameInfoPath); err == nil {
			return absGogGameInfoPath, nil
		}
	} else {
		return "", err
	}

	return "", nil
}

func macOsFindBundleApp(id, langCode string, rdx redux.Readable) (string, error) {

	absInstalledPath, err := osInstalledPath(id, vangogh_integration.MacOS, langCode, rdx)
	if err != nil {
		return "", err
	}

	if strings.HasSuffix(absInstalledPath, appBundleExt) {
		return absInstalledPath, nil
	}

	var matches []string
	if matches, err = filepath.Glob(filepath.Join(absInstalledPath, "*"+appBundleExt)); err == nil {
		if len(matches) == 1 {
			return matches[0], nil
		}
	}

	return "", errors.New("cannot locate macOS bundle.app for " + id)
}

func macOsExecTaskGogGameInfo(absGogGameInfoPath string, gogGameInfo *gog_integration.GogGameInfo, et *execTask) (*execTask, error) {

	pt, err := gogGameInfo.GetPlayTask(et.playTask)
	if err != nil {
		return nil, err
	}

	absGogGameInfoDir, _ := filepath.Split(absGogGameInfoPath)
	absExeRootDir := strings.TrimSuffix(absGogGameInfoDir, relMacOsGogGameInfoDir+"/")

	exePath := pt.Path
	// account for Windows-style relative paths, e.g. DOSBOX\DOSBOX.exe
	if parts := strings.Split(exePath, "\\"); len(parts) > 1 {
		exePath = filepath.Join(parts...)
	}

	absExePath := filepath.Join(absExeRootDir, exePath)

	et.name = pt.Name
	et.exe = absExePath
	et.workDir = filepath.Join(absExeRootDir, pt.WorkingDir)

	if pt.Arguments != "" {
		et.args = append(et.args, pt.Arguments)
	}

	return et, nil
}

func macOsExecTaskBundleApp(absBundleAppPath string, et *execTask) (*execTask, error) {

	et.exe = "open"
	et.args = append([]string{absBundleAppPath}, et.args...)

	return et, nil
}
