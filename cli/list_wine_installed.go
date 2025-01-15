package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"net/url"
)

func ListWineInstalledHandler(u *url.URL) error {
	size := u.Query().Has("size")
	langCode := defaultLangCode
	if u.Query().Has(vangogh_integration.LanguageCodeProperty) {
		langCode = u.Query().Get(vangogh_integration.LanguageCodeProperty)
	}
	return ListInstalled(vangogh_integration.Windows, langCode, size)
}
