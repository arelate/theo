package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

func WineUpdateHandler(u *url.URL) error {

	q := u.Query()
	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	all := q.Has("all")
	reveal := q.Has("reveal")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.ServerConnectionProperties, vangogh_integration.TitleProperty, vangogh_integration.SlugProperty, data.InstallParametersProperty)
	if err != nil {
		return err
	}

	return Update(vangogh_integration.Windows, langCode, rdx, all, reveal, ids...)
}
