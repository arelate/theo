package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path/filepath"
)

func RevealInstalledHandler(u *url.URL) error {

	ids := Ids(u)

	langCode := defaultLangCode
	if u.Query().Has(vangogh_integration.LanguageCodeProperty) {
		langCode = u.Query().Get(vangogh_integration.LanguageCodeProperty)
	}

	return RevealInstalled(langCode, ids...)
}

func RevealInstalled(langCode string, ids ...string) error {

	fia := nod.NewProgress("revealing installed products...")
	defer fia.EndWithResult("done")

	currentOs := []vangogh_integration.OperatingSystem{data.CurrentOs()}
	langCodes := []string{langCode}

	vangogh_integration.PrintParams(ids, currentOs, langCodes, nil, true)

	fia.TotalInt(len(ids))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return fia.EndWithError(err)
	}

	rdx, err := redux.NewReader(reduxDir, data.ServerConnectionProperties, data.BundleNameProperty)
	if err != nil {
		return fia.EndWithError(err)
	}

	return currentOsRevealInstalledApps(langCode, rdx, ids...)
}

func currentOsRevealInstalledApps(langCode string, rdx redux.Readable, ids ...string) error {

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	if len(ids) == 1 {
		bundleName, _ := rdx.GetLastVal(data.BundleNameProperty, ids[0])
		productInstalledAppDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(data.CurrentOs(), langCode), bundleName)
		return currentOsReveal(productInstalledAppDir)
	} else {
		return currentOsReveal(installedAppsDir)
	}
}
