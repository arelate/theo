package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const (
	catCmdPfx = "cat "
)

func macOsInstallProduct(id string,
	productDetails *vangogh_integration.ProductDetails,
	link *vangogh_integration.ProductDownloadLink,
	rdx redux.Writeable,
	force bool) error {

	mia := nod.Begin("installing %s version of %s...", vangogh_integration.MacOS, productDetails.Title)
	defer mia.Done()

	if err := macOsExtractInstaller(id, link, force); err != nil {
		return err
	}

	if err := macOsPlaceExtracts(id, link, rdx, force); err != nil {
		return err
	}

	if err := macOsPostInstallActions(id, link, rdx); err != nil {
		return err
	}

	if err := macOsRemoveProductExtracts(id, productDetails); err != nil {
		return err
	}

	return nil
}

func macOsExtractInstaller(id string, link *vangogh_integration.ProductDownloadLink, force bool) error {

	meia := nod.Begin(" extracting installer with pkgutil, please wait...")
	defer meia.Done()

	if data.CurrentOs() != vangogh_integration.MacOS {
		return errors.New("extracting .pkg installers is only supported on " + vangogh_integration.MacOS.String())
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	extractsDir, err := pathways.GetAbsRelDir(data.MacOsExtracts)
	if err != nil {
		return err
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)
	productExtractsDir := filepath.Join(extractsDir, id)

	localFilenameExtractsDir := filepath.Join(productExtractsDir, link.LocalFilename)
	// if the product extracts dir already exists - that would imply that the product
	// has been extracted already. Remove the directory with contents if forced
	// Return early otherwise (if not forced).
	if _, err := os.Stat(localFilenameExtractsDir); err == nil {
		if force {
			if err := os.RemoveAll(localFilenameExtractsDir); err != nil {
				return err
			}
		} else {
			return nil
		}
	}

	productExtractDir, _ := filepath.Split(localFilenameExtractsDir)
	if _, err := os.Stat(productExtractDir); os.IsNotExist(err) {
		if err := os.MkdirAll(productExtractDir, 0755); err != nil {
			return err
		}
	}

	localDownload := filepath.Join(productDownloadsDir, link.LocalFilename)

	cmd := exec.Command("pkgutil", "--expand-full", localDownload, localFilenameExtractsDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func macOsPlaceExtracts(id string, link *vangogh_integration.ProductDownloadLink, rdx redux.Writeable, force bool) error {

	mpea := nod.Begin(" placing product installation files...")
	defer mpea.Done()

	if data.CurrentOs() != vangogh_integration.MacOS {
		return errors.New("placing .pkg extracts is only supported on " + vangogh_integration.MacOS.String())
	}

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	extractsDir, err := pathways.GetAbsRelDir(data.MacOsExtracts)
	if err != nil {
		return err
	}

	productExtractsDir := filepath.Join(extractsDir, id)

	absPostInstallScriptPath := PostInstallScriptPath(productExtractsDir, link)
	postInstallScript, err := ParsePostInstallScript(absPostInstallScriptPath)
	if err != nil {
		return err
	}

	absExtractPayloadPath := filepath.Join(productExtractsDir, link.LocalFilename, relPayloadPath)

	if _, err = os.Stat(absExtractPayloadPath); os.IsNotExist(err) {
		return errors.New("cannot locate extracts payload")
	}

	installerType := postInstallScript.InstallerType()

	absBundlePath, err := data.GetAbsBundlePath(id, link.LanguageCode, vangogh_integration.MacOS, rdx)

	if strings.HasSuffix(postInstallScript.bundleName, data.MacOsAppBundleExt) {
		absBundlePath = filepath.Join(absBundlePath, postInstallScript.bundleName)
	}

	switch installerType {
	case "game":
		return macOsPlaceGame(absExtractPayloadPath, absBundlePath, force)
	case "dlc":
		return macOsPlaceDlc(absExtractPayloadPath, absBundlePath, force)
	default:
		return errors.New("unknown postinstall script installer type: " + installerType)
	}
}

func macOsPlaceGame(absExtractsPayloadPath, absInstallationPath string, force bool) error {

	mpga := nod.Begin(" placing game installation files...")
	defer mpga.Done()

	// when installing a game
	if _, err := os.Stat(absInstallationPath); err == nil {
		if force {
			if err := os.RemoveAll(absInstallationPath); err != nil {
				return err
			}
		} else {
			// already installed, overwrite won't be forced
			return nil
		}
	}

	installationDir, _ := filepath.Split(absInstallationPath)
	if _, err := os.Stat(installationDir); os.IsNotExist(err) {
		if err := os.MkdirAll(installationDir, 0755); err != nil {
			return err
		}
	}

	return os.Rename(absExtractsPayloadPath, absInstallationPath)
}

func macOsPlaceDlc(absExtractsPayloadPath, absInstallationPath string, force bool) error {

	mpda := nod.Begin(" placing downloadable content files...")
	defer mpda.Done()

	if _, err := os.Stat(absInstallationPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absInstallationPath, 0755); err != nil {
			return err
		}
	}

	// enumerate all DLC files in the payload directory
	dlcFiles := make([]string, 0)

	if err := filepath.Walk(absExtractsPayloadPath, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if relPath, err := filepath.Rel(absExtractsPayloadPath, path); err == nil {
				dlcFiles = append(dlcFiles, relPath)
			} else {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	for _, dlcFile := range dlcFiles {

		absDstPath := filepath.Join(absInstallationPath, dlcFile)
		absDstDir, _ := filepath.Split(absDstPath)

		if _, err := os.Stat(absDstDir); os.IsNotExist(err) {
			if err := os.MkdirAll(absDstDir, 0755); err != nil {
				return err
			}
		}

		absSrcPath := filepath.Join(absExtractsPayloadPath, dlcFile)

		if err := os.Rename(absSrcPath, absDstPath); err != nil {
			return err
		}
	}

	return nil
}

func macOsPostInstallActions(id string,
	link *vangogh_integration.ProductDownloadLink,
	rdx redux.Readable) error {

	mpia := nod.Begin(" performing post-install %s actions for %s...", vangogh_integration.MacOS, id)
	defer mpia.Done()

	if filepath.Ext(link.LocalFilename) != pkgExt {
		// for macOS - there's nothing to be done for additional files (that are not .pkg installers)
		return nil
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)

	extractsDir, err := pathways.GetAbsRelDir(data.MacOsExtracts)
	if err != nil {
		return err
	}

	productExtractsDir := filepath.Join(extractsDir, id)

	absPostInstallScriptPath := PostInstallScriptPath(productExtractsDir, link)

	pis, err := ParsePostInstallScript(absPostInstallScriptPath)
	if err != nil {
		return err
	}

	absBundlePath, err := data.GetAbsBundlePath(id, link.LanguageCode, vangogh_integration.MacOS, rdx)

	// macOS bundle path points to a directory, not an .app package
	// try to locate .app package inside the bundle dir
	if !strings.HasSuffix(absBundlePath, data.MacOsAppBundleExt) {
		absBundlePath = macOsLocateAppBundle(absBundlePath)
	}

	if customCommands := pis.CustomCommands(); len(customCommands) > 0 {
		if err := macOsProcessPostInstallScript(customCommands, productDownloadsDir, absBundlePath); err != nil {
			return err
		}
	}

	if err := macOsRemoveXattrs(absBundlePath); err != nil {
		return err
	}

	return nil
}

func macOsLocateAppBundle(path string) string {

	if strings.HasSuffix(path, data.MacOsAppBundleExt) {
		return path
	}

	if matches, err := filepath.Glob(filepath.Join(path, "*"+data.MacOsAppBundleExt)); err == nil {
		if len(matches) == 1 {
			return matches[0]
		}
	}

	return path
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

func macOsProcessPostInstallScript(commands []string, productDownloadsDir, bundleInstallPath string) error {

	pcca := nod.NewProgress(" processing post-install commands...")
	defer pcca.Done()

	pcca.TotalInt(len(commands))

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, catCmdPfx) {
			if catCmdParts := strings.Split(strings.TrimPrefix(cmd, catCmdPfx), " "); len(catCmdParts) == 3 {
				srcGlob := strings.Trim(strings.Replace(catCmdParts[0], "\"${pkgpath}\"", productDownloadsDir, 1), "\"")
				dstPath := strings.Trim(strings.Replace(catCmdParts[2], "${gog_full_path}", bundleInstallPath, 1), "\"")
				if err := macOsExecCatFiles(srcGlob, dstPath); err != nil {
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

func macOsExecCatFiles(srcGlob string, dstPath string) error {

	if srcGlob == "" {
		return errors.New("cat command source glob cannot be empty")
	}
	if dstPath == "" {
		return errors.New("cat command destination path cannot be empty")
	}

	_, srcFileGlob := filepath.Split(srcGlob)

	ecfa := nod.NewProgress(" cat %s into %s...", srcFileGlob, dstPath)
	defer ecfa.Done()

	dstDir, _ := filepath.Split(dstPath)
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dstDir, 0755); err != nil {
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

func macOsRemoveProductExtracts(id string, productDetails *vangogh_integration.ProductDetails) error {

	rela := nod.Begin(" removing extracts for %s...", productDetails.Title)
	defer rela.Done()

	extractsDir, err := pathways.GetAbsRelDir(data.MacOsExtracts)
	if err != nil {
		return err
	}

	idPath := filepath.Join(extractsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rela.EndWithResult("product extracts dir not present")
		return nil
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(vangogh_integration.MacOS)

	for _, dl := range dls {

		path := filepath.Join(extractsDir, id, dl.LocalFilename)

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fa.EndWithResult("not present")
			continue
		}

		if err := os.RemoveAll(path); err != nil {
			return err
		}

		fa.Done()
	}

	rdda := nod.Begin(" removing empty product extracts directory...")
	if err := removeDirIfEmpty(idPath); err != nil {
		return err
	}
	rdda.Done()

	return nil
}

func hasOnlyDSStore(entries []fs.DirEntry) bool {
	if len(entries) == 1 {
		return entries[0].Name() == ".DS_Store"
	}
	return false
}

func isDirEmpty(dirPath string) (bool, error) {
	if entries, err := os.ReadDir(dirPath); err == nil && len(entries) == 0 {
		return true, nil
	} else if err == nil && hasOnlyDSStore(entries) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	return false, nil
}

func removeDirIfEmpty(dirPath string) error {
	if empty, err := isDirEmpty(dirPath); empty && err == nil {
		return os.RemoveAll(dirPath)
	} else if err != nil {
		return err
	}
	return nil
}

func macOsReveal(path string) error {
	cmd := exec.Command("open", "-R", path)
	return cmd.Run()
}

func macOsExecute(path string, env []string, verbose bool) error {

	path = macOsLocateAppBundle(path)

	cmd := exec.Command("open", path)
	cmd.Dir = path

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	for _, e := range env {
		cmd.Env = append(cmd.Env, e)
	}

	return cmd.Run()
}
