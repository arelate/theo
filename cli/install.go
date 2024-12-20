package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	pkgExt = ".pkg"
	exeExt = ".exe"
	shExt  = ".sh"
)

const (
	relPayloadPath = "package.pkg/Scripts/payload"
)

func InstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	_, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	keepDownloads := q.Has("keep-downloads")
	force := q.Has("force")

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	return Install(ids, langCode, downloadTypes, keepDownloads, force)
}

func Install(ids []string,
	langCode string,
	downloadTypes []vangogh_local_data.DownloadType,
	keepDownloads bool,
	force bool) error {

	ia := nod.Begin("installing products...")
	defer ia.EndWithResult("done")

	currentOs := []vangogh_local_data.OperatingSystem{CurrentOS()}
	langCodes := []string{langCode}

	vangogh_local_data.PrintParams(ids, currentOs, langCodes, downloadTypes, true)

	if err := BackupMetadata(); err != nil {
		return err
	}

	if err := Download(ids, currentOs, langCodes, downloadTypes, force); err != nil {
		return err
	}

	if err := Validate(ids, currentOs, langCodes); err != nil {
		return err
	}

	if err := PinInstalledMetadata(ids, force); err != nil {
		return err
	}

	if err := currentOsInstall(ids, langCode, downloadTypes, force); err != nil {
		return err
	}

	if !keepDownloads {
		if err := RemoveDownloads(ids, currentOs, langCodes, force); err != nil {
			return err
		}
	}

	if err := RevealInstalled(ids, langCode); err != nil {
		return err
	}

	return nil
}

func currentOsInstall(ids []string,
	langCode string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	ia := nod.NewProgress("installing products...")
	defer ia.EndWithResult("done")

	currentOs := []vangogh_local_data.OperatingSystem{CurrentOS()}
	langCodes := []string{langCode}

	vangogh_local_data.PrintParams(ids, currentOs, langCodes, downloadTypes, true)

	ia.TotalInt(len(ids))

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return ia.EndWithError(err)
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return ia.EndWithError(err)
	}

	for _, id := range ids {

		if metadata, err := GetDownloadMetadata(id, currentOs, langCodes, downloadTypes, force); err == nil {

			for _, link := range metadata.DownloadLinks {
				linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
				linkExt := filepath.Ext(link.LocalFilename)
				absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

				if linkOs != CurrentOS() {
					continue
				}

				switch linkOs {
				case vangogh_local_data.MacOS:
					extractsDir, err := pathways.GetAbsRelDir(data.Extracts)
					if err != nil {
						return ia.EndWithError(err)
					}

					if linkExt == pkgExt {
						if err := macOsInstall(id, metadata, &link, downloadsDir, extractsDir, installedAppsDir, force); err != nil {
							return ia.EndWithError(err)
						}
					}
				case vangogh_local_data.Windows:
					if linkExt == exeExt {
						if err := windowsInstall(id, &link, absInstallerPath, installedAppsDir); err != nil {
							return ia.EndWithError(err)
						}
					}
				case vangogh_local_data.Linux:
					if linkExt == shExt {
						if err := linuxInstall(id, &link, absInstallerPath, installedAppsDir); err != nil {
							return ia.EndWithError(err)
						}
					}
				default:
					return ia.EndWithError(errors.New("unknown os" + linkOs.String()))
				}
			}

		} else {
			return ia.EndWithError(err)
		}

		ia.Increment()
	}

	return nil
}

func macOsInstall(id string,
	metadata *vangogh_local_data.DownloadMetadata,
	link *vangogh_local_data.DownloadLink,
	downloadsDir, extractsDir, installedAppsDir string,
	force bool) error {

	productDownloadsDir := filepath.Join(downloadsDir, id)
	productExtractsDir := filepath.Join(extractsDir, id)
	osLangInstalledAppsDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(vangogh_local_data.MacOS, link.LanguageCode))

	if err := macOsExtractInstaller(link, productDownloadsDir, productExtractsDir, force); err != nil {
		return err
	}

	if err := macOsPlaceExtracts(link, productExtractsDir, osLangInstalledAppsDir, force); err != nil {
		return err
	}

	if err := macOsPostInstallActions(id, link, installedAppsDir); err != nil {
		return err
	}

	if err := macOsRemoveProductExtracts(id, metadata, extractsDir); err != nil {
		return err
	}

	return nil
}

func linuxInstall(id string,
	link *vangogh_local_data.DownloadLink,
	absInstallerPath, installedAppsDir string) error {

	if _, err := os.Stat(absInstallerPath); err != nil {
		return err
	}

	if err := linuxPostDownloadActions(id, link); err != nil {
		return err
	}

	productInstalledDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(vangogh_local_data.Linux, link.LanguageCode))

	// https://www.reddit.com/r/linux_gaming/comments/42l258/fully_automated_gog_games_install_howto/
	cmd := exec.Command(absInstallerPath, "--", "--i-agree-to-all-licenses", "--noreadme", "--nooptions", "--noprompt", "--destination", productInstalledDir)
	return cmd.Run()
}

func windowsInstall(id string,
	link *vangogh_local_data.DownloadLink,
	absInstallerPath, installedAppsDir string) error {

	if CurrentOS() != vangogh_local_data.Windows {
		return errors.New("Windows install is only supported on Windows, use wine-install to install Windows version on " + CurrentOS().String())
	}

	return errors.New("native Windows installation is not implemented")
}
