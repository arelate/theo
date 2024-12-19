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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	catCmdPfx = "cat "
)

func PostInstallHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, _ := OsLangCodeDownloadType(u)

	return PostInstall(ids, operatingSystems, langCodes)
}

func PostInstall(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string) error {

	fia := nod.NewProgress("performing post install actions...")
	defer fia.EndWithResult("done")

	vangogh_local_data.PrintParams(ids, operatingSystems, nil, nil, true)

	fia.TotalInt(len(ids))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return fia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir, data.SetupProperties, data.BundleNameProperty)
	if err != nil {
		return fia.EndWithError(err)
	}

	installerDownloadType := []vangogh_local_data.DownloadType{vangogh_local_data.Installer}

	for _, id := range ids {

		if metadata, err := GetDownloadMetadata(id, operatingSystems, langCodes, installerDownloadType, false); err == nil {

			for _, link := range metadata.DownloadLinks {

				linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
				if linkOs == vangogh_local_data.MacOS {
					if err := postMacOsProductInstall(id, &link, rdx); err != nil {
						return fia.EndWithError(err)
					}
				}
			}

		} else {
			return fia.EndWithError(err)
		}

		fia.Increment()
	}

	return nil
}

func postMacOsProductInstall(id string,
	link *vangogh_local_data.DownloadLink,
	rdx kevlar.WriteableRedux) error {

	pmia := nod.Begin(" performing post install macOS actions...")
	defer pmia.EndWithResult("done")

	if filepath.Ext(link.LocalFilename) != pkgExt {
		// for macOS - there's nothing to be done for additional files (that are not .pkg installers)
		return nil
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return pmia.EndWithError(err)
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)

	extractsDir, err := pathways.GetAbsDir(data.Extracts)
	if err != nil {
		return pmia.EndWithError(err)
	}

	productExtractsDir := filepath.Join(extractsDir, id)

	installationDir := defaultInstallationDir
	if setupInstallDir, ok := rdx.GetLastVal(data.SetupProperties, data.InstallationPathProperty); ok && setupInstallDir != "" {
		installationDir = setupInstallDir
	}

	absPostInstallScriptPath := PostInstallScriptPath(productExtractsDir, link)

	pis, err := ParsePostInstallScript(absPostInstallScriptPath)
	if err != nil {
		return pmia.EndWithError(err)
	}

	bundleName := pis.BundleName()

	// storing bundle name to use later
	if err := rdx.AddValues(data.BundleNameProperty, id, bundleName); err != nil {
		return pmia.EndWithError(err)
	}

	bundleInstallPath := filepath.Join(installationDir, bundleName)

	if customCommands := pis.CustomCommands(); len(customCommands) > 0 {
		if err := processPostInstallScript(customCommands, productDownloadsDir, bundleInstallPath); err != nil {
			return pmia.EndWithError(err)
		}
	}

	if err := removeXattrs(bundleInstallPath); err != nil {
		return pmia.EndWithError(err)
	}

	if err := codeSign(bundleInstallPath); err != nil {
		return pmia.EndWithError(err)
	}

	return nil
}

func removeXattrs(path string) error {

	// xattr -cr /Applications/Bundle Name.app
	cmd := exec.Command("xattr", "-cr", path)
	return cmd.Run()
}

func codeSign(path string) error {
	// codesign --force --deep --sign - /Applications/Bundle Name.app
	cmd := exec.Command("codesign", "--force", "--deep", "--sign", "-", path)
	return cmd.Run()
}

func processPostInstallScript(commands []string, productDownloadsDir, bundleInstallPath string) error {

	pcca := nod.NewProgress(" processing custom commands...")
	defer pcca.EndWithResult("done")

	pcca.TotalInt(len(commands))

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, catCmdPfx) {
			if catCmdParts := strings.Split(strings.TrimPrefix(cmd, catCmdPfx), " "); len(catCmdParts) == 3 {
				srcGlob := strings.Trim(strings.Replace(catCmdParts[0], "\"${pkgpath}\"", productDownloadsDir, 1), "\"")
				dstPath := strings.Trim(strings.Replace(catCmdParts[2], "${gog_full_path}", bundleInstallPath, 1), "\"")
				if err := execCatFiles(srcGlob, dstPath); err != nil {
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

func execCatFiles(srcGlob string, dstPath string) error {

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
