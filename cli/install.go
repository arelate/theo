package cli

import (
	"github.com/arelate/vangogh_local_data"
	"net/url"
	"runtime"
	"strings"
)

func InstallHandler(u *url.URL) error {

	q := u.Query()

	var ids []string
	if q.Has(vangogh_local_data.IdProperty) {
		ids = strings.Split(q.Get(vangogh_local_data.IdProperty), ",")
	}

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

	force := q.Has("force")

	return Install(ids, operatingSystems, langCodes, downloadTypes, force)
}

func Install(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {
	return nil
}
