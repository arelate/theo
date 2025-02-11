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

const (
	macOsAppBundleExt = ".app"
)

func macOsInstallProduct(id string,
	metadata *vangogh_integration.TheoMetadata,
	link *vangogh_integration.TheoDownloadLink,
	downloadsDir, extractsDir, installedAppsDir string,
	rdx redux.Writeable,
	force bool) error {

	mia := nod.Begin("installing %s version of %s...", vangogh_integration.MacOS, metadata.Title)
	defer mia.EndWithResult("done")

	productDownloadsDir := filepath.Join(downloadsDir, id)
	productExtractsDir := filepath.Join(extractsDir, id)
	osLangInstalledAppsDir := filepath.Join(installedAppsDir, data.OsLangCode(vangogh_integration.MacOS, link.LanguageCode))

	if err := macOsExtractInstaller(link, productDownloadsDir, productExtractsDir, force); err != nil {
		return mia.EndWithError(err)
	}

	if err := macOsPlaceExtracts(id, link, productExtractsDir, osLangInstalledAppsDir, rdx, force); err != nil {
		return mia.EndWithError(err)
	}

	if err := macOsPostInstallActions(id, link, installedAppsDir); err != nil {
		return mia.EndWithError(err)
	}

	if err := macOsRemoveProductExtracts(id, metadata, extractsDir); err != nil {
		return mia.EndWithError(err)
	}

	return nil
}

func macOsExtractInstaller(link *vangogh_integration.TheoDownloadLink, productDownloadsDir, productExtractsDir string, force bool) error {

	meia := nod.Begin(" extracting installer with pkgutil, please wait...")
	defer meia.EndWithResult("done")

	if data.CurrentOs() != vangogh_integration.MacOS {
		return meia.EndWithError(errors.New("extracting .pkg installers is only supported on " + vangogh_integration.MacOS.String()))
	}

	localFilenameExtractsDir := filepath.Join(productExtractsDir, link.LocalFilename)
	// if the product extracts dir already exists - that would imply that the product
	// has been extracted already. Remove the directory with contents if forced
	// Return early otherwise (if not forced).
	if _, err := os.Stat(localFilenameExtractsDir); err == nil {
		if force {
			if err := os.RemoveAll(localFilenameExtractsDir); err != nil {
				return meia.EndWithError(err)
			}
		} else {
			return nil
		}
	}

	productExtractDir, _ := filepath.Split(localFilenameExtractsDir)
	if _, err := os.Stat(productExtractDir); os.IsNotExist(err) {
		if err := os.MkdirAll(productExtractDir, 0755); err != nil {
			return meia.EndWithError(err)
		}
	}

	localDownload := filepath.Join(productDownloadsDir, link.LocalFilename)

	cmd := exec.Command("pkgutil", "--expand-full", localDownload, localFilenameExtractsDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func macOsPlaceExtracts(id string, link *vangogh_integration.TheoDownloadLink, productExtractsDir, osLangInstalledAppsDir string, rdx redux.Writeable, force bool) error {

	mpea := nod.Begin(" placing product installation files...")
	defer mpea.EndWithResult("done")

	if data.CurrentOs() != vangogh_integration.MacOS {
		return mpea.EndWithError(errors.New("placing .pkg extracts is only supported on " + vangogh_integration.MacOS.String()))
	}

	if err := rdx.MustHave(data.BundleNameProperty); err != nil {
		return mpea.EndWithError(err)
	}

	absPostInstallScriptPath := PostInstallScriptPath(productExtractsDir, link)
	postInstallScript, err := ParsePostInstallScript(absPostInstallScriptPath)
	if err != nil {
		return mpea.EndWithError(err)
	}

	absExtractPayloadPath := filepath.Join(productExtractsDir, link.LocalFilename, relPayloadPath)

	if _, err := os.Stat(absExtractPayloadPath); os.IsNotExist(err) {
		return mpea.EndWithError(errors.New("cannot locate extracts payload"))
	}

	bundleName := postInstallScript.BundleName()

	if bundleName == "" {
		return mpea.EndWithError(errors.New("cannot determine bundle name from postinstall file"))
	}

	if err := rdx.AddValues(data.BundleNameProperty, id, bundleName); err != nil {
		return mpea.EndWithError(err)
	}

	installerType := postInstallScript.InstallerType()
	absBundlePath := filepath.Join(osLangInstalledAppsDir, bundleName)

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
	defer mpga.EndWithResult("done")

	// when installing a game
	if _, err := os.Stat(absInstallationPath); err == nil {
		if force {
			if err := os.RemoveAll(absInstallationPath); err != nil {
				return mpga.EndWithError(err)
			}
		} else {
			// already installed, overwrite won't be forced
			return nil
		}
	}

	installationDir, _ := filepath.Split(absInstallationPath)
	if _, err := os.Stat(installationDir); os.IsNotExist(err) {
		if err := os.MkdirAll(installationDir, 0755); err != nil {
			return mpga.EndWithError(err)
		}
	}

	return os.Rename(absExtractsPayloadPath, absInstallationPath)
}

func macOsPlaceDlc(absExtractsPayloadPath, absInstallationPath string, force bool) error {

	mpda := nod.Begin(" placing downloadable content files...")
	defer mpda.EndWithResult("done")

	if _, err := os.Stat(absInstallationPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absInstallationPath, 0755); err != nil {
			return mpda.EndWithError(err)
		}
	}

	// enumerate all DLC files in the payload directory
	dlcFiles := make([]string, 0)

	if err := filepath.Walk(absExtractsPayloadPath, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if relPath, err := filepath.Rel(absExtractsPayloadPath, path); err == nil {
				dlcFiles = append(dlcFiles, relPath)
			} else {
				return mpda.EndWithError(err)
			}
		}
		return nil
	}); err != nil {
		return mpda.EndWithError(err)
	}

	for _, dlcFile := range dlcFiles {

		absDstPath := filepath.Join(absInstallationPath, dlcFile)
		absDstDir, _ := filepath.Split(absDstPath)

		if _, err := os.Stat(absDstDir); os.IsNotExist(err) {
			if err := os.MkdirAll(absDstDir, 0755); err != nil {
				return mpda.EndWithError(err)
			}
		}

		absSrcPath := filepath.Join(absExtractsPayloadPath, dlcFile)

		if err := os.Rename(absSrcPath, absDstPath); err != nil {
			return mpda.EndWithError(err)
		}
	}

	return nil
}

func macOsPostInstallActions(id string,
	link *vangogh_integration.TheoDownloadLink,
	installedAppsDir string) error {

	mpia := nod.Begin(" performing post-install %s actions for %s...", vangogh_integration.MacOS, id)
	defer mpia.EndWithResult("done")

	if filepath.Ext(link.LocalFilename) != pkgExt {
		// for macOS - there's nothing to be done for additional files (that are not .pkg installers)
		return nil
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return mpia.EndWithError(err)
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)

	extractsDir, err := pathways.GetAbsRelDir(data.MacOsExtracts)
	if err != nil {
		return mpia.EndWithError(err)
	}

	productExtractsDir := filepath.Join(extractsDir, id)

	absPostInstallScriptPath := PostInstallScriptPath(productExtractsDir, link)

	pis, err := ParsePostInstallScript(absPostInstallScriptPath)
	if err != nil {
		return mpia.EndWithError(err)
	}

	bundleName := pis.BundleName()

	absBundlePath := filepath.Join(installedAppsDir, data.OsLangCode(vangogh_integration.MacOS, link.LanguageCode), bundleName)

	// some macOS bundles point to a directory, not an .app package
	// try to locate .app package inside the bundle dir
	if !strings.HasSuffix(absBundlePath, macOsAppBundleExt) {
		absBundlePath = macOsLocateAppBundle(absBundlePath)
	}

	if customCommands := pis.CustomCommands(); len(customCommands) > 0 {
		if err := macOsProcessPostInstallScript(customCommands, productDownloadsDir, absBundlePath); err != nil {
			return mpia.EndWithError(err)
		}
	}

	if err := macOsRemoveXattrs(absBundlePath); err != nil {
		return mpia.EndWithError(err)
	}

	return nil
}

func macOsLocateAppBundle(path string) string {

	if strings.HasSuffix(path, macOsAppBundleExt) {
		return path
	}

	if matches, err := filepath.Glob(filepath.Join(path, "*"+macOsAppBundleExt)); err == nil {
		if len(matches) == 1 {
			return matches[0]
		}
	}

	return path
}

func macOsRemoveXattrs(path string) error {

	mrxa := nod.Begin(" removing xattrs...")
	defer mrxa.EndWithResult("done")

	// xattr -cr /Applications/Bundle Name.app
	cmd := exec.Command("xattr", "-cr", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func macOsProcessPostInstallScript(commands []string, productDownloadsDir, bundleInstallPath string) error {

	pcca := nod.NewProgress(" processing post-install commands...")
	defer pcca.EndWithResult("done")

	pcca.TotalInt(len(commands))

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, catCmdPfx) {
			if catCmdParts := strings.Split(strings.TrimPrefix(cmd, catCmdPfx), " "); len(catCmdParts) == 3 {
				srcGlob := strings.Trim(strings.Replace(catCmdParts[0], "\"${pkgpath}\"", productDownloadsDir, 1), "\"")
				dstPath := strings.Trim(strings.Replace(catCmdParts[2], "${gog_full_path}", bundleInstallPath, 1), "\"")
				if err := macOsExecCatFiles(srcGlob, dstPath); err != nil {
					return pcca.EndWithError(err)
				}
			}
			pcca.Increment()
			continue
		}
		// at this point we've handled all known commands, so anything here would be unknown
		return pcca.EndWithError(errors.New("cannot process unknown custom command: " + cmd))
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
	defer ecfa.EndWithResult("done")

	dstDir, _ := filepath.Split(dstPath)
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return ecfa.EndWithError(err)
		}
	}

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return ecfa.EndWithError(err)
	}
	defer dstFile.Close()

	matches, err := filepath.Glob(srcGlob)
	if err != nil {
		return ecfa.EndWithError(err)
	}

	slices.Sort(matches)

	ecfa.TotalInt(len(matches))

	for _, match := range matches {

		srcFile, err := os.Open(match)
		if err != nil {
			return ecfa.EndWithError(err)
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			_ = srcFile.Close()
			return ecfa.EndWithError(err)
		}

		ecfa.Increment()
		_ = srcFile.Close()

		if err := os.Remove(match); err != nil {
			return ecfa.EndWithError(err)
		}
	}

	return nil
}

func macOsRemoveProductExtracts(id string,
	metadata *vangogh_integration.TheoMetadata,
	extractsDir string) error {

	rela := nod.Begin(" removing extracts for %s...", metadata.Title)
	defer rela.EndWithResult("done")

	idPath := filepath.Join(extractsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rela.EndWithResult("product extracts dir not present")
		return nil
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(vangogh_integration.MacOS)

	for _, dl := range dls {

		path := filepath.Join(extractsDir, id, dl.LocalFilename)

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fa.EndWithResult("not present")
			continue
		}

		if err := os.RemoveAll(path); err != nil {
			return fa.EndWithError(err)
		}

		fa.EndWithResult("done")
	}

	rdda := nod.Begin(" removing empty product extracts directory...")
	if err := removeDirIfEmpty(idPath); err != nil {
		return rdda.EndWithError(err)
	}
	rdda.EndWithResult("done")

	return nil
}

func hasOnlyDSStore(entries []fs.DirEntry) bool {
	if len(entries) == 1 {
		return entries[0].Name() == ".DS_Store"
	}
	return false
}

func removeDirIfEmpty(dirPath string) error {
	if entries, err := os.ReadDir(dirPath); err == nil && len(entries) == 0 {
		if err := os.Remove(dirPath); err != nil {
			return err
		}
	} else if err == nil && hasOnlyDSStore(entries) {
		if err := os.RemoveAll(dirPath); err != nil {
			return err
		}
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

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	for _, e := range env {
		cmd.Env = append(cmd.Env, e)
	}

	return cmd.Run()
}
