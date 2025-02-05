package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"os"
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
	force := q.Has("force")

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	ip := &installParameters{
		operatingSystem: data.CurrentOs(),
		langCode:        langCode,
		downloadTypes:   downloadTypes,
		keepDownloads:   q.Has("keep-downloads"),
		noSteamShortcut: q.Has("no-steam-shortcut"),
	}

	return Install(ip, force, ids...)
}

func Install(ip *installParameters, force bool, ids ...string) error {

	ia := nod.Begin("installing products...")
	defer ia.EndWithResult("done")

	currentOs := []vangogh_integration.OperatingSystem{data.CurrentOs()}
	langCodes := []string{ip.langCode}

	vangogh_integration.PrintParams(ids, currentOs, langCodes, ip.downloadTypes, true)

	supported, err := filterNotSupported(ip.langCode, force, ids...)
	if err != nil {
		return ia.EndWithError(err)
	}

	if len(supported) > 0 {
		ids = supported
	} else {
		ia.EndWithResult("requested products are not supported on %s", data.CurrentOs())
		return nil
	}

	notInstalled, err := filterNotInstalled(ip.langCode, ids...)
	if err != nil {
		return ia.EndWithError(err)
	}

	if len(notInstalled) > 0 {
		if !force {
			ids = notInstalled
		}
	} else if !force {
		ia.EndWithResult("all requested products are already installed")
		return nil
	}

	if err = BackupMetadata(); err != nil {
		return ia.EndWithError(err)
	}

	if err = Download(currentOs, langCodes, ip.downloadTypes, force, ids...); err != nil {
		return ia.EndWithError(err)
	}

	if err = Validate(currentOs, langCodes, ip.downloadTypes, ids...); err != nil {
		return ia.EndWithError(err)
	}

	for _, id := range ids {
		if err := currentOsInstallProduct(id, ip.langCode, ip.downloadTypes, force); err != nil {
			return ia.EndWithError(err)
		}
	}

	if !ip.noSteamShortcut {
		if err := AddSteamShortcut(ip.langCode, runLaunchOptionsTemplate, force, ids...); err != nil {
			return ia.EndWithError(err)
		}
	}

	if !ip.keepDownloads {
		if err = RemoveDownloads(currentOs, langCodes, ip.downloadTypes, force, ids...); err != nil {
			return ia.EndWithError(err)
		}
	}

	if err = pinInstalledMetadata(currentOs, ip.langCode, force, ids...); err != nil {
		return ia.EndWithError(err)
	}

	if err = pinInstallParameters(ip, ids...); err != nil {
		return ia.EndWithError(err)
	}

	if err = RevealInstalled(ip.langCode, ids...); err != nil {
		return ia.EndWithError(err)
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

	osLangCodeDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(data.CurrentOs(), langCode))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return nil, fia.EndWithError(err)
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.BundleNameProperty)
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

func filterNotSupported(langCode string, force bool, ids ...string) ([]string, error) {

	fnsa := nod.NewProgress(" checking operating systems support...")
	defer fnsa.EndWithResult("done")

	fnsa.TotalInt(len(ids))

	supported := make([]string, 0, len(ids))

	for _, id := range ids {

		metadata, err := getTheoMetadata(id, force)
		if err != nil {
			return nil, fnsa.EndWithError(err)
		}

		dls := metadata.DownloadLinks.
			FilterOperatingSystems(data.CurrentOs()).
			FilterLanguageCodes(langCode).
			FilterDownloadTypes(vangogh_integration.Installer)

		if len(dls) > 0 {
			supported = append(supported, id)
		}

		fnsa.Increment()
	}

	return supported, nil
}

func currentOsInstallProduct(id string, langCode string, downloadTypes []vangogh_integration.DownloadType, force bool) error {

	coipa := nod.Begin(" installing %s on %s...", id, data.CurrentOs())
	defer coipa.EndWithResult("done")

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return coipa.EndWithError(err)
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return coipa.EndWithError(err)
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return coipa.EndWithError(err)
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.BundleNameProperty)
	if err != nil {
		return coipa.EndWithError(err)
	}

	metadata, err := getTheoMetadata(id, force)
	if err != nil {
		return coipa.EndWithError(err)
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(data.CurrentOs()).
		FilterLanguageCodes(langCode).
		FilterDownloadTypes(downloadTypes...)

	if len(dls) == 0 {
		coipa.EndWithResult("no links are matching operating params")
		return nil
	}

	for _, link := range dls {

		linkOs := vangogh_integration.ParseOperatingSystem(link.OS)
		linkExt := filepath.Ext(link.LocalFilename)
		absInstallerPath := filepath.Join(downloadsDir, id, link.LocalFilename)

		switch linkOs {
		case vangogh_integration.MacOS:
			extractsDir, err := pathways.GetAbsRelDir(data.MacOsExtracts)
			if err != nil {
				return coipa.EndWithError(err)
			}

			if linkExt == pkgExt {
				if err := macOsInstallProduct(id, metadata, &link, downloadsDir, extractsDir, installedAppsDir, rdx, force); err != nil {
					return coipa.EndWithError(err)
				}
			}
		case vangogh_integration.Linux:
			if linkExt == shExt {
				if err := linuxInstallProduct(id, metadata, &link, absInstallerPath, installedAppsDir, rdx); err != nil {
					return coipa.EndWithError(err)
				}
			}
		case vangogh_integration.Windows:
			if linkExt == exeExt {
				if err := windowsInstallProduct(id, metadata, &link, absInstallerPath, installedAppsDir); err != nil {
					return coipa.EndWithError(err)
				}
			}
		default:
			return coipa.EndWithError(errors.New("unknown os" + linkOs.String()))
		}
	}
	return nil
}
