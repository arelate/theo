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
	"strings"
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

	currentOs := []vangogh_local_data.OperatingSystem{data.CurrentOS()}
	langCodes := []string{langCode}

	vangogh_local_data.PrintParams(ids, currentOs, langCodes, downloadTypes, true)

	if notInstalled, err := filterNotInstalled(langCode, ids...); err == nil {

		if len(notInstalled) > 0 {
			ids = notInstalled
		} else if !force {
			return nil
		}

	} else {
		return err
	}

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

func filterNotInstalled(langCode string, ids ...string) ([]string, error) {

	fia := nod.Begin(" checking existing installations...")
	defer fia.EndWithResult("done")

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return nil, fia.EndWithError(err)
	}

	osLangCodeDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(data.CurrentOS(), langCode))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return nil, fia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir, data.SlugProperty, data.BundleNameProperty)
	if err != nil {
		return nil, fia.EndWithError(err)
	}

	notInstalled := make([]string, 0, len(ids))

	for _, id := range ids {

		if bundleName, ok := rdx.GetLastVal(data.BundleNameProperty, id); ok && bundleName != "" {

			bundlePath := filepath.Join(osLangCodeDir, bundleName)
			if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
				notInstalled = append(notInstalled, id)
			}

		} else {
			notInstalled = append(notInstalled, id)
		}
	}

	if len(notInstalled) == 0 {
		fia.EndWithResult("products have existing installations: %s", strings.Join(ids, ","))
	} else {
		fia.EndWithResult(
			"%d product require installation: %s",
			len(notInstalled),
			strings.Join(notInstalled, ","))
	}

	return notInstalled, nil
}

func currentOsInstall(ids []string,
	langCode string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	ia := nod.NewProgress("installing products...")
	defer ia.EndWithResult("done")

	currentOs := []vangogh_local_data.OperatingSystem{data.CurrentOS()}
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

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return ia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir, data.SlugProperty, data.BundleNameProperty)
	if err != nil {
		return ia.EndWithError(err)
	}

	for _, id := range ids {

		if metadata, err := LoadOrFetchTheoMetadata(id, currentOs, langCodes, downloadTypes, force); err == nil {

			for _, link := range metadata.DownloadLinks {
				linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
				linkExt := filepath.Ext(link.LocalFilename)
				absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

				if linkOs != data.CurrentOS() {
					continue
				}

				switch linkOs {
				case vangogh_local_data.MacOS:
					extractsDir, err := pathways.GetAbsRelDir(data.MacOsExtracts)
					if err != nil {
						return ia.EndWithError(err)
					}

					if linkExt == pkgExt {
						if err := macOsInstall(id, metadata, &link, downloadsDir, extractsDir, installedAppsDir, rdx, force); err != nil {
							return ia.EndWithError(err)
						}
					}
				case vangogh_local_data.Linux:
					if linkExt == shExt {
						if err := linuxInstall(id, metadata, &link, absInstallerPath, installedAppsDir, rdx); err != nil {
							return ia.EndWithError(err)
						}
					}
				case vangogh_local_data.Windows:
					if linkExt == exeExt {
						if err := windowsInstall(id, metadata, &link, absInstallerPath, installedAppsDir); err != nil {
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
	metadata *vangogh_local_data.TheoMetadata,
	link *vangogh_local_data.DownloadLink,
	downloadsDir, extractsDir, installedAppsDir string,
	rdx kevlar.WriteableRedux,
	force bool) error {

	mia := nod.Begin("installing macOS version of %s...", metadata.Title)
	defer mia.EndWithResult("done")

	productDownloadsDir := filepath.Join(downloadsDir, id)
	productExtractsDir := filepath.Join(extractsDir, id)
	osLangInstalledAppsDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(vangogh_local_data.MacOS, link.LanguageCode))

	if err := macOsExtractInstaller(link, productDownloadsDir, productExtractsDir, force); err != nil {
		return err
	}

	if err := macOsPlaceExtracts(id, link, productExtractsDir, osLangInstalledAppsDir, rdx, force); err != nil {
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
	metadata *vangogh_local_data.TheoMetadata,
	link *vangogh_local_data.DownloadLink,
	absInstallerPath, installedAppsDir string,
	rdx kevlar.WriteableRedux) error {

	lia := nod.Begin("installing Linux version of %s...")
	defer lia.EndWithResult("done")

	if err := rdx.MustHave(data.SlugProperty, data.BundleNameProperty); err != nil {
		return err
	}

	if _, err := os.Stat(absInstallerPath); err != nil {
		return err
	}

	if err := linuxPostDownloadActions(id, link); err != nil {
		return err
	}

	productTitle, _ := rdx.GetLastVal(data.SlugProperty, id)

	if err := rdx.ReplaceValues(data.BundleNameProperty, id, productTitle); err != nil {
		return err
	}

	productInstalledAppDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(vangogh_local_data.Linux, link.LanguageCode), productTitle)

	// https://www.reddit.com/r/linux_gaming/comments/42l258/fully_automated_gog_games_install_howto/
	cmd := exec.Command(absInstallerPath, "--", "--i-agree-to-all-licenses", "--noreadme", "--nooptions", "--noprompt", "--destination", productInstalledAppDir)
	return cmd.Run()
}

func windowsInstall(id string,
	metadata *vangogh_local_data.TheoMetadata,
	link *vangogh_local_data.DownloadLink,
	absInstallerPath, installedAppsDir string) error {

	wia := nod.Begin("installing Windows version of %s...", metadata.Title)
	defer wia.EndWithResult("done")

	return errors.New("Windows installation is not implemented")
}
