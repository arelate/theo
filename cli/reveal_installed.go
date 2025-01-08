package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

func RevealInstalledHandler(u *url.URL) error {

	ids := Ids(u)

	langCode := defaultLangCode
	if u.Query().Has(vangogh_local_data.LanguageCodeProperty) {
		langCode = u.Query().Get(vangogh_local_data.LanguageCodeProperty)
	}

	return RevealInstalled(ids, langCode)
}

func RevealInstalled(ids []string, langCode string) error {

	fia := nod.NewProgress("revealing installed products...")
	defer fia.EndWithResult("done")

	currentOs := []vangogh_local_data.OperatingSystem{data.CurrentOS()}
	langCodes := []string{langCode}

	vangogh_local_data.PrintParams(ids, currentOs, langCodes, nil, true)

	fia.TotalInt(len(ids))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return fia.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties, data.BundleNameProperty)
	if err != nil {
		return fia.EndWithError(err)
	}

	return currentOsRevealInstalledApps(langCode, rdx, ids...)
}

func currentOsRevealInstalledApps(langCode string, rdx kevlar.ReadableRedux, ids ...string) error {

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return currentOsReveal(installedAppsDir)
	}

	for _, id := range ids {
		bundleName, _ := rdx.GetLastVal(data.BundleNameProperty, id)
		productInstalledAppDir := filepath.Join(installedAppsDir, data.OsLangCodeDir(data.CurrentOS(), langCode), bundleName)
		if err := currentOsReveal(productInstalledAppDir); err != nil {
			return err
		}
	}

	return nil
}
