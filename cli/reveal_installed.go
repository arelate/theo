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

	ids := Ids(u)

	langCode := defaultLangCode
	if u.Query().Has(vangogh_integration.LanguageCodeProperty) {
		langCode = u.Query().Get(vangogh_integration.LanguageCodeProperty)
	}

	return RevealInstalled(langCode, ids...)
}

func RevealInstalled(langCode string, ids ...string) error {

	fia := nod.NewProgress("revealing installed products...")
	defer fia.Done()

	fia.TotalInt(len(ids))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir, data.ServerConnectionProperties, data.BundleNameProperty)
	if err != nil {
		return err
	}

	return currentOsRevealInstalledApps(langCode, rdx, ids...)
}

func currentOsRevealInstalledApps(langCode string, rdx redux.Readable, ids ...string) error {
	var revealPath string
	var err error

	switch len(ids) {
	case 1:
		revealPath, err = data.GetAbsBundlePath(ids[0], langCode, data.CurrentOs(), rdx)
	default:
		revealPath, err = pathways.GetAbsDir(data.InstalledApps)
	}

	if err != nil {
		return err
	}

	return currentOsReveal(revealPath)

}
