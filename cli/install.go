package cli

import (
	"errors"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"
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

	id := q.Get(vangogh_integration.IdProperty)

	os := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		os = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	var downloadTypes []vangogh_integration.DownloadType
	if q.Has(vangogh_integration.DownloadTypeProperty) {
		dts := strings.Split(q.Get(vangogh_integration.DownloadTypeProperty), ",")
		downloadTypes = vangogh_integration.ParseManyDownloadTypes(dts)
	} else {
		downloadTypes = append(downloadTypes, vangogh_integration.Installer, vangogh_integration.DLC)
	}

	ii := &InstallInfo{
		OperatingSystem: os,
		LangCode:        langCode,
		DownloadTypes:   downloadTypes,
		KeepDownloads:   q.Has("keep-downloads"),
		NoSteamShortcut: q.Has("no-steam-shortcut"),
		reveal:          q.Has("reveal"),
		verbose:         q.Has("verbose"),
		force:           q.Has("force"),
	}

	if q.Has("env") {
		ii.Env = strings.Split(q.Get("env"), ",")
	}

	return Install(id, ii)
}

func Install(id string, ii *InstallInfo) error {

	ia := nod.Begin("installing %s...", id)
	defer ia.Done()

	operatingSystems := []vangogh_integration.OperatingSystem{ii.OperatingSystem}
	langCodes := []string{ii.LangCode}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	vangogh_integration.PrintParams([]string{id}, operatingSystems, langCodes, ii.DownloadTypes, true)

	productDetails, err := GetProductDetails(id, rdx, ii.force)
	if err != nil {
		return err
	}

	switch productDetails.ProductType {
	case vangogh_integration.DlcProductType:
		ia.EndWithResult("install %s required product(s) to get this downloadable content", strings.Join(productDetails.RequiresGames, ","))
		return nil
	case vangogh_integration.PackProductType:
		ia.EndWithResult("installing product(s) included in this pack: %s", strings.Join(productDetails.IncludesGames, ","))
		for _, includedId := range productDetails.IncludesGames {
			if err = Install(includedId, ii); err != nil {
				return err
			}
		}
		return nil
	case vangogh_integration.GameProductType:
		// do nothing
	default:
		return errors.New("unknown product type " + productDetails.ProductType)
	}

	if ii.OperatingSystem == vangogh_integration.AnyOperatingSystem {
		if slices.Contains(productDetails.OperatingSystems, data.CurrentOs()) {
			ii.OperatingSystem = data.CurrentOs()
		} else if slices.Contains(productDetails.OperatingSystems, vangogh_integration.Windows) {
			ii.OperatingSystem = vangogh_integration.Windows
		} else {
			unsupportedOsMsg := fmt.Sprintf("product doesn't support %s or %s, only %v",
				data.CurrentOs(), vangogh_integration.Windows, productDetails.OperatingSystems)
			return errors.New(unsupportedOsMsg)
		}
	}

	ii.AddProductDetails(productDetails)

	if !ii.force {

		if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

			installInfo, err := matchInstallInfo(ii, installedInfoLines...)
			if err != nil {
				return err
			}

			if installInfo != nil {
				ia.EndWithResult("product %s is already installed", id)
				return nil
			} else {
				return err
			}

		}

	}

	if err = BackupMetadata(); err != nil {
		return err
	}

	os := []vangogh_integration.OperatingSystem{ii.OperatingSystem}

	if err = Download(os, langCodes, ii.DownloadTypes, nil, rdx, ii.force, id); err != nil {
		return err
	}

	if err = Validate(os, langCodes, ii.DownloadTypes, nil, rdx, id); err != nil {
		return err
	}

	if err = osInstallProduct(id, ii, productDetails, rdx); err != nil {
		return err
	}

	if !ii.NoSteamShortcut {
		if err = AddSteamShortcut(id, ii.OperatingSystem, ii.LangCode, rdx, ii.force); err != nil {
			return err
		}
	}

	if !ii.KeepDownloads {
		if err = RemoveDownloads(os, langCodes, ii.DownloadTypes, rdx, ii.force, id); err != nil {
			return err
		}
	}

	if err = pinInstallInfo(id, ii, rdx); err != nil {
		return err
	}

	idInstalledDate := map[string][]string{id: {time.Now().UTC().Format(time.RFC3339)}}
	if err = rdx.BatchReplaceValues(data.InstallDateProperty, idInstalledDate); err != nil {
		return err
	}

	if ii.reveal {
		if err = RevealInstalled(id, ii); err != nil {
			return err
		}
	}

	return nil
}

func osInstallProduct(id string, ii *InstallInfo, productDetails *vangogh_integration.ProductDetails, rdx redux.Writeable) error {

	start := time.Now().UTC().Unix()

	coipa := nod.Begin("installing %s, %s, %s...", id, ii.OperatingSystem, ii.LangCode)
	defer coipa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		coipa.EndWithResult("no links are matching install params")
		return nil
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	if err = hasFreeSpaceForProduct(productDetails, installedAppsDir,
		[]vangogh_integration.OperatingSystem{ii.OperatingSystem}, []string{ii.LangCode}, ii.DownloadTypes, nil, ii.force); err != nil {
		return err
	}

	installerExts := []string{pkgExt, shExt, exeExt}

	for _, link := range dls {

		if !slices.Contains(installerExts, filepath.Ext(link.LocalFilename)) {
			continue
		}

		switch link.OperatingSystem {
		case vangogh_integration.MacOS:

			if err = macOsInstallProduct(id, productDetails, &link, rdx, ii.force); err != nil {
				return err
			}

		case vangogh_integration.Linux:

			if err = linuxInstallProduct(id, productDetails, &link, rdx); err != nil {
				return err
			}

		case vangogh_integration.Windows:

			switch data.CurrentOs() {
			case vangogh_integration.Windows:

				if err = windowsInstallProduct(id, productDetails, &link, rdx); err != nil {
					return err
				}

			default:

				if err = prefixInit(id, ii.LangCode, rdx, ii.verbose); err != nil {
					return err
				}

				if err = prefixInstallProduct(id, ii.LangCode, rdx, ii.Env, ii.DownloadTypes, ii.verbose, ii.force); err != nil {
					return err
				}

				if err = prefixCreateInventory(id, ii.LangCode, rdx, start); err != nil {
					return err
				}

				if err = DefaultPrefixEnv(ii.LangCode, id); err != nil {
					return err
				}

			}
		default:
			return link.OperatingSystem.ErrUnsupported()
		}
	}

	return nil
}
