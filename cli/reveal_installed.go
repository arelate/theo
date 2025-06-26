package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

func RevealInstalledHandler(u *url.URL) error {

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

	return RevealInstalled(id, ii)
}

func RevealInstalled(id string, ii *InstallInfo) error {

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
		os, err := installedInfoOperatingSystem(id, rdx)
		if err != nil {
			return err
		}

		ii.OperatingSystem = os
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
