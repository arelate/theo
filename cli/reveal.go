package cli

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func RevealHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	langCode := "" // installed info language will be used instead of default
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
	}

	installed := q.Has("installed")
	downloads := q.Has("downloads")
	backups := q.Has("backups")

	return Reveal(id, ii, installed, downloads, backups)
}

func Reveal(id string, ii *InstallInfo, installed, downloads, backups bool) error {

	if installed {
		if err := revealInstalled(id, ii); err != nil {
			return err
		}
	}

	if downloads {
		if ii.SteamInstall {
			return errors.New("revealing Steam downloads is not supported")
		} else {
			if err := revealDownloads(id); err != nil {
				return err
			}
		}
	}

	if backups {
		if err := revealBackups(); err != nil {
			return err
		}
	}

	return nil
}

func revealInstalled(id string, ii *InstallInfo) error {

	ria := nod.Begin("revealing installation for %s...", id)
	defer ria.Done()

	rdx, err := redux.NewReader(data.AbsReduxDir(),
		data.InstallInfoProperty,
		vangogh_integration.TitleProperty,
		vangogh_integration.SlugProperty)
	if err != nil {
		return err
	}

	if ii.OperatingSystem == vangogh_integration.AnyOperatingSystem {
		iios, err := installedInfoOperatingSystem(id, rdx)
		if err != nil {
			return err
		}

		ii.OperatingSystem = iios
	}

	if ii.LangCode == "" && !ii.SteamInstall {
		var lc string
		lc, err = installedInfoLangCode(id, ii.OperatingSystem, rdx)
		if err != nil {
			return err
		}

		ii.LangCode = lc
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		var installedInfo *InstallInfo
		installedInfo, _, err = matchInstallInfo(ii, installedInfoLines...)
		if err != nil {
			return err
		}

		if installedInfo != nil {
			ii.SteamInstall = installedInfo.SteamInstall
			return currentOsRevealInstalled(id, ii, rdx)
		} else {
			ria.EndWithResult("no install found for %s %s-%s", id, ii.OperatingSystem, ii.LangCode)
		}

	}

	return nil
}

func currentOsRevealInstalled(id string, ii *InstallInfo, rdx redux.Readable) error {

	var revealPath string
	var err error

	switch ii.SteamInstall {
	case true:
		revealPath, err = data.AbsSteamAppInstallDir(id, ii.OperatingSystem, rdx)
	default:
		revealPath, err = osInstalledPath(id, ii, rdx)
	}

	if err != nil {
		return err
	}

	return currentOsReveal(revealPath)
}

func revealBackups() error {

	rda := nod.Begin("revealing backups...")
	defer rda.Done()

	return currentOsReveal(data.Pwd.AbsDirPath(data.Backups))
}

func revealDownloads(id string) error {

	rda := nod.Begin("revealing downloads...")
	defer rda.Done()

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	productDownloadsDir := filepath.Join(downloadsDir, id)

	if _, err := os.Stat(productDownloadsDir); err == nil {
		return currentOsReveal(productDownloadsDir)
	} else {
		return currentOsReveal(downloadsDir)
	}
}

func currentOsReveal(path string) error {
	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		return macOsReveal(path)
	case vangogh_integration.Windows:
		return windowsReveal(path)
	case vangogh_integration.Linux:
		return linuxReveal(path)
	default:
		return errors.New("cannot reveal on unknown operating system")
	}
}
