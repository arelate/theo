package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"golang.org/x/exp/slices"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	catCmdPfx = "cat "
)

func macOsExtractInstaller(link *vangogh_local_data.DownloadLink, productDownloadsDir, productExtractsDir string, force bool) error {

	if CurrentOS() != vangogh_local_data.MacOS {
		return errors.New("extracting .pkg installers is only supported on macOS")
	}

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

	return cmd.Run()
}

func macOsPlaceExtracts(id string, link *vangogh_local_data.DownloadLink, productExtractsDir, osLangInstalledAppsDir string, rdx kevlar.WriteableRedux, force bool) error {

	if CurrentOS() != vangogh_local_data.MacOS {
		return errors.New("placing .pkg extracts is only supported on macOS")
	}

	if err := rdx.MustHave(data.BundleNameProperty); err != nil {
		return err
	}

	absPostInstallScriptPath := PostInstallScriptPath(productExtractsDir, link)
	postInstallScript, err := ParsePostInstallScript(absPostInstallScriptPath)
	if err != nil {
		return err
	}

	absExtractPayloadPath := filepath.Join(productExtractsDir, link.LocalFilename, relPayloadPath)

	if _, err := os.Stat(absExtractPayloadPath); os.IsNotExist(err) {
		return errors.New("cannot locate extracts payload")
	}

	bundleName := postInstallScript.BundleName()

	if bundleName == "" {
		return errors.New("cannot determine bundle name from postinstall file")
	}

	if err := rdx.AddValues(data.BundleNameProperty, id, bundleName); err != nil {
		return err
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

func macOsPostDownloadActions(id, path string) error {

	if CurrentOS() != vangogh_local_data.MacOS {
		return errors.New("macOS post-download actions are only supported on macOS")
	}

	mpda := nod.Begin(" performing macOS post-download actions for %s...", id)
	defer mpda.EndWithResult("done")

	return macOsRemoveXattrs(path)
}

func macOsPostInstallActions(id string,
	link *vangogh_local_data.DownloadLink,
	installedAppsDir string) error {

	mpia := nod.Begin(" performing post-install macOS actions for %s...", id)
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

	absBundlePath := filepath.Join(installedAppsDir, data.OsLangCodeDir(vangogh_local_data.MacOS, link.LanguageCode), bundleName)

	if customCommands := pis.CustomCommands(); len(customCommands) > 0 {
		if err := macOsProcessPostInstallScript(customCommands, productDownloadsDir, absBundlePath); err != nil {
			return mpia.EndWithError(err)
		}
	}

	if err := macOsRemoveXattrs(absBundlePath); err != nil {
		return mpia.EndWithError(err)
	}

	if err := macOsCodeSign(absBundlePath); err != nil {
		return mpia.EndWithError(err)
	}

	return nil
}

func macOsRemoveXattrs(path string) error {

	// xattr -cr /Applications/Bundle Name.app
	cmd := exec.Command("xattr", "-cr", path)
	return cmd.Run()
}

func macOsCodeSign(path string) error {
	// codesign --force --deep --sign - /Applications/Bundle Name.app
	cmd := exec.Command("codesign", "--force", "--deep", "--sign", "-", path)
	return cmd.Run()
}

func macOsProcessPostInstallScript(commands []string, productDownloadsDir, bundleInstallPath string) error {

	pcca := nod.NewProgress(" processing custom commands...")
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
	metadata *vangogh_local_data.DownloadMetadata,
	extractsDir string) error {

	rela := nod.Begin(" removing extracts for %s...", metadata.Title)
	defer rela.EndWithResult("done")

	idPath := filepath.Join(extractsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rela.EndWithResult("product extracts dir not present")
		return nil
	}

	for _, dl := range metadata.DownloadLinks {

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
