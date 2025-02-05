package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
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

	return Update(vangogh_integration.Windows, langCode, all, ids...)
}
