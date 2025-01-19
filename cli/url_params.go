package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"net/url"
	"strings"
)

const defaultLangCode = "en"

func Ids(u *url.URL) []string {

	q := u.Query()

	var ids []string
	if q.Has(vangogh_integration.IdProperty) {
		ids = strings.Split(q.Get(vangogh_integration.IdProperty), ",")
	}

	return ids
}

func OsLangCodeDownloadType(u *url.URL) ([]vangogh_integration.OperatingSystem, []string, []vangogh_integration.DownloadType) {

	q := u.Query()

	operatingSystems := vangogh_integration.OperatingSystemsFromUrl(u)
	if len(operatingSystems) == 0 {
		operatingSystems = append(operatingSystems, data.CurrentOs())
	}

	var langCodes []string
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCodes = strings.Split(q.Get(vangogh_integration.LanguageCodeProperty), ",")
	}

	if len(langCodes) == 0 {
		langCodes = append(langCodes, defaultLangCode)
	}

	downloadTypes := vangogh_integration.DownloadTypesFromUrl(u)
	if len(downloadTypes) == 0 {
		downloadTypes = append(downloadTypes, vangogh_integration.Installer, vangogh_integration.DLC)
	}

	return operatingSystems, langCodes, downloadTypes
}
