package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"strings"
)

func WineUpdateHandler(u *url.URL) error {

	q := u.Query()
	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	var env []string
	if q.Has("env") {
		env = strings.Split(q.Get("env"), ",")
	}
	verbose := q.Has("verbose")

	all := q.Has("all")
	reveal := q.Has("reveal")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return WineUpdate(langCode, rdx, env, verbose, reveal, all, ids...)
}

func WineUpdate(langCode string, rdx redux.Writeable, env []string, verbose, reveal, all bool, ids ...string) error {
	wua := nod.NewProgress("updating installed %s products...", vangogh_integration.Windows.String())
	defer wua.Done()

	updatedIds, err := filterUpdatedProducts(vangogh_integration.Windows, langCode, rdx, all, ids...)
	if err != nil {
		return err
	}

	for _, id := range updatedIds {
		ip, err := loadInstallParameters(id, vangogh_integration.Windows, langCode, rdx, reveal, true)
		if err != nil {
			return err
		}

		if err = WineInstall(ip, env, verbose, id); err != nil {
			return err
		}
	}

	return nil
}
