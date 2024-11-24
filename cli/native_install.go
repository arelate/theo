package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

func NativeInstallHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return NativeInstall(ids, operatingSystems, langCodes, downloadTypes, force)
}

func NativeInstall(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	nia := nod.NewProgress("running native OS installation methods...")
	defer nia.EndWithResult("done")

	vangogh_local_data.PrintParams(ids, operatingSystems, langCodes, downloadTypes)

	nia.TotalInt(len(ids))

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return nia.EndWithError(err)
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return nia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties)
	if err != nil {
		return nia.EndWithError(err)
	}

	installationDir := defaultInstallationDir
	if setupInstallDir, ok := rdx.GetLastVal(data.SetupProperties, data.InstallationPathProperty); ok && setupInstallDir != "" {
		installationDir = setupInstallDir
	}

	for _, id := range ids {

		if title, links, err := GetTitleDownloadLinks(id, operatingSystems, langCodes, downloadTypes, force); err == nil {
			if err = nativeProductInstall(id, title, downloadsDir, installationDir, links); err != nil {
				return nia.EndWithError(err)
			}
		} else {
			return nia.EndWithError(err)
		}

		nia.Increment()
	}

	return nil
}

func nativeProductInstall(id, title string, downloadsDir, installationDir string, links []vangogh_local_data.DownloadLink) error {
	npia := nod.Begin(" natively installing %s...", title)
	defer npia.EndWithResult("done")

	productDownloadsDir := filepath.Join(downloadsDir, id)

	for _, link := range links {
		linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
		linkExt := filepath.Ext(link.LocalFilename)
		localFilePath := filepath.Join(productDownloadsDir, link.LocalFilename)

		if linkOs == vangogh_local_data.MacOS && linkExt == pkgExt {
			if err := nativeMacOsInstall(localFilePath); err != nil {
				return npia.EndWithError(err)
			}
		}
		if linkOs == vangogh_local_data.Windows && linkExt == exeExt {
			if err := nativeWindowsInstall(localFilePath, installationDir); err != nil {
				return npia.EndWithError(err)
			}
		}
		if linkOs == vangogh_local_data.Linux && linkExt == shExt {
			if err := nativeLinuxInstall(localFilePath, installationDir); err != nil {
				return npia.EndWithError(err)
			}
		}
	}

	return nil
}

func nativeMacOsInstall(installerPath string) error {

	if _, err := os.Stat(installerPath); err != nil {
		return err
	}

	if err := removeXattrs(installerPath); err != nil {
		return err
	}

	// using CurrentUserHomeDirectory to avoid the need to run as root
	// macOS installers produced by GOG will ask for installation location anyway
	cmd := exec.Command("installer", "-pkg", installerPath, "-target", "CurrentUserHomeDirectory")
	return cmd.Run()
}

func nativeWindowsInstall(installerPath, installationDir string) error {
	return errors.New("native Windows installation is not implemented")
}

func nativeLinuxInstall(installerPath, installationDir string) error {
	return errors.New("native Linux installation is not implemented")
}
