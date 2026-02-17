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
		switch ii.Origin {
		case data.VangoghGogOrigin:
			if err := revealDownloads(id); err != nil {
				return err
			}
		default:
			return errors.New("downloads reveal not supported for " + ii.Origin.String())
		}
	}

	if backups {
		if err := revealBackups(); err != nil {
			return err
		}
	}

	return nil
}

func revealInstalled(id string, request *InstallInfo) error {

	ria := nod.Begin("revealing installation for %s...", id)
	defer ria.Done()

	rdx, err := redux.NewReader(data.AbsReduxDir(),
		data.InstallInfoProperty,
		vangogh_integration.TitleProperty,
		vangogh_integration.SlugProperty)
	if err != nil {
		return err
	}

	ii, err := matchInstalledInfo(id, request, rdx)
	if err != nil {
		return err
	}

	return currentOsRevealInstalled(id, ii, rdx)
}

func currentOsRevealInstalled(id string, ii *InstallInfo, rdx redux.Readable) error {

	var revealPath string
	var err error

	switch ii.Origin {
	case data.VangoghGogOrigin:
		revealPath, err = originOsInstalledPath(id, ii, rdx)
	case data.SteamOrigin:
		revealPath, err = data.AbsSteamAppInstallDir(id, ii.OperatingSystem, rdx)
	default:
		return ii.Origin.ErrUnsupportedOrigin()
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
