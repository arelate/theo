package cli

import (
	"fmt"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"strings"
)

func PrintParams(
	ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType) {

	ppa := nod.Begin("resolved parameters =")
	defer ppa.End()

	sb := strings.Builder{}

	params := make(map[string][]string)

	for _, id := range ids {
		params[vangogh_local_data.IdProperty] = append(params[vangogh_local_data.IdProperty], id)
	}

	for _, os := range operatingSystems {
		params[vangogh_local_data.OperatingSystemsProperty] = append(params[vangogh_local_data.OperatingSystemsProperty], os.String())
	}

	for _, lc := range langCodes {
		params[vangogh_local_data.LanguageCodeProperty] = append(params[vangogh_local_data.LanguageCodeProperty], lc)
	}

	for _, dt := range downloadTypes {
		params["download-type"] = append(params["download-type"], dt.String())
	}

	for _, p := range []string{vangogh_local_data.IdProperty, vangogh_local_data.OperatingSystemsProperty, vangogh_local_data.LanguageCodeProperty, "download-type"} {
		if _, ok := params[p]; !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s: %s; ", p, strings.Join(params[p], ", ")))
	}

	ppa.EndWithResult(sb.String())
}
