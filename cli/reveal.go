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
		if err := revealDownloads(id); err != nil {
			return err
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

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir,
		data.InstallInfoProperty,
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

	if ii.LangCode == "" {
		lc, err := installedInfoLangCode(id, ii.OperatingSystem, rdx)
		if err != nil {
			return err
		}

		ii.LangCode = lc
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		installedInfo, err := matchInstallInfo(ii, installedInfoLines...)
		if err != nil {
			return err
		}

		if installedInfo != nil {
			return currentOsRevealInstalled(id, ii, rdx)
		} else {
			ria.EndWithResult("no installation found for %s-%s", ii.OperatingSystem, ii.LangCode)
		}

	}

	return nil
}

func currentOsRevealInstalled(id string, ii *InstallInfo, rdx redux.Readable) error {

	revealPath, err := osInstalledPath(id, ii.OperatingSystem, ii.LangCode, rdx)
	if err != nil {
		return err
	}

	return currentOsReveal(revealPath)
}

func revealBackups() error {

	rda := nod.Begin("revealing backups...")
	defer rda.Done()

	backupsDir, err := pathways.GetAbsDir(data.Backups)
	if err != nil {
		return err
	}

	return currentOsReveal(backupsDir)
}

func revealDownloads(id string) error {

	rda := nod.Begin("revealing downloads...")
	defer rda.Done()

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)

	if _, err = os.Stat(productDownloadsDir); err == nil {
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
