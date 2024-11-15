package cli

import (
	"github.com/arelate/vangogh_local_data"
	"net/url"
	"runtime"
	"strings"
)

func Ids(u *url.URL) []string {

	q := u.Query()

	var ids []string
	if q.Has(vangogh_local_data.IdProperty) {
		ids = strings.Split(q.Get(vangogh_local_data.IdProperty), ",")
	}

	return ids
}

func OsLangCodeDownloadType(u *url.URL) ([]vangogh_local_data.OperatingSystem, []string, []vangogh_local_data.DownloadType) {

	q := u.Query()

	operatingSystems := vangogh_local_data.OperatingSystemsFromUrl(u)
	if len(operatingSystems) == 0 {
		switch runtime.GOOS {
		case "windows":
			operatingSystems = append(operatingSystems, vangogh_local_data.Windows)
		case "darwin":
			operatingSystems = append(operatingSystems, vangogh_local_data.MacOS)
		case "linux":
			operatingSystems = append(operatingSystems, vangogh_local_data.Windows)
		}
	}

	var langCodes []string
	if q.Has(vangogh_local_data.LanguageCodeProperty) {
		langCodes = strings.Split(q.Get(vangogh_local_data.LanguageCodeProperty), ",")
	}

	if len(langCodes) == 0 {
		langCodes = append(langCodes, defaultLangCode)
	}

	downloadTypes := []vangogh_local_data.DownloadType{vangogh_local_data.Installer}

	if !q.Has("no-dlc") {
		downloadTypes = append(downloadTypes, vangogh_local_data.DLC)
	}

	return operatingSystems, langCodes, downloadTypes
}
