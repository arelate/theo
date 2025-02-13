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
	"slices"
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
	defer ia.Done()

	currentOs := []vangogh_integration.OperatingSystem{data.CurrentOs()}
	langCodes := []string{ip.langCode}

	vangogh_integration.PrintParams(ids, currentOs, langCodes, ip.downloadTypes, true)
	if err := resolveProductTitles(ids...); err != nil {
		return err
	}

	supported, err := filterNotSupported(ip.langCode, force, ids...)
	if err != nil {
		return err
	}

	if len(supported) > 0 {
		ids = supported
	} else {
		ia.EndWithResult("requested products are not supported on %s", data.CurrentOs())
		return nil
	}

	notInstalled, err := filterNotInstalled(ip.langCode, ids...)
	if err != nil {
		return err
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
		return err
	}

	if err = Download(currentOs, langCodes, ip.downloadTypes, force, ids...); err != nil {
		return err
	}

	if err = Validate(currentOs, langCodes, ip.downloadTypes, ids...); err != nil {
		return err
	}

	for _, id := range ids {
		if err = currentOsInstallProduct(id, ip.langCode, ip.downloadTypes, force); err != nil {
			return err
		}
	}

	if !ip.noSteamShortcut {
		if err := AddSteamShortcut(ip.langCode, runLaunchOptionsTemplate, force, ids...); err != nil {
			return err
		}
	}

	if !ip.keepDownloads {
		if err = RemoveDownloads(currentOs, langCodes, ip.downloadTypes, force, ids...); err != nil {
			return err
		}
	}

	if err = pinInstalledMetadata(currentOs, ip.langCode, force, ids...); err != nil {
		return err
	}

	if err = pinInstallParameters(ip, ids...); err != nil {
		return err
	}

	if err = RevealInstalled(ip.langCode, ids...); err != nil {
		return err
	}

	return nil
}

func filterNotInstalled(langCode string, ids ...string) ([]string, error) {

	fia := nod.Begin(" checking existing installations...")
	defer fia.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return nil, err
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.BundleNameProperty)
	if err != nil {
		return nil, err
	}

	notInstalled := make([]string, 0, len(ids))

	for _, id := range ids {

		if absBundlePath, err := data.GetAbsBundlePath(id, langCode, data.CurrentOs(), rdx); err == nil {
			if _, err := os.Stat(absBundlePath); os.IsNotExist(err) {
				notInstalled = append(notInstalled, id)
			}
		} else {
			notInstalled = append(notInstalled, id)
			continue
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
	defer fnsa.Done()

	fnsa.TotalInt(len(ids))

	supported := make([]string, 0, len(ids))

	for _, id := range ids {

		metadata, err := getTheoMetadata(id, force)
		if err != nil {
			return nil, err
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
	defer coipa.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.BundleNameProperty)
	if err != nil {
		return err
	}

	metadata, err := getTheoMetadata(id, force)
	if err != nil {
		return err
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(data.CurrentOs()).
		FilterLanguageCodes(langCode).
		FilterDownloadTypes(downloadTypes...)

	if len(dls) == 0 {
		coipa.EndWithResult("no links are matching operating params")
		return nil
	}

	installerExts := []string{pkgExt, shExt, exeExt}

	for _, link := range dls {

		if !slices.Contains(installerExts, filepath.Ext(link.LocalFilename)) {
			continue
		}

		switch vangogh_integration.ParseOperatingSystem(link.OS) {
		case vangogh_integration.MacOS:
			if err = macOsInstallProduct(id, metadata, &link, rdx, force); err != nil {
				return err
			}
		case vangogh_integration.Linux:
			if err = linuxInstallProduct(id, metadata, &link, rdx); err != nil {
				return err
			}
		case vangogh_integration.Windows:
			if err = windowsInstallProduct(id, metadata, &link, rdx); err != nil {
				return err
			}
		default:
			return errors.New("unknown os" + link.OS)
		}
	}
	return nil
}
