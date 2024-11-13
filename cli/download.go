package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"strings"
)

const defaultLangCode = "en"

func DownloadHandler(u *url.URL) error {

	q := u.Query()

	var ids []string
	if q.Has(vangogh_local_data.IdProperty) {
		ids = strings.Split(q.Get(vangogh_local_data.IdProperty), ",")
	}

	operatingSystems := vangogh_local_data.OperatingSystemsFromUrl(u)
	if len(operatingSystems) == 0 {
		operatingSystems = append(operatingSystems, vangogh_local_data.MacOS)
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

	return Download(ids, operatingSystems, langCodes, downloadTypes, force)
}

func Download(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	da := nod.NewProgress("downloading game data from vangogh...")
	defer da.End()

	dmd, err := pathways.GetAbsRelDir(data.DownloadsMetadata)
	if err != nil {
		return da.EndWithError(err)
	}

	kvdm, err := kevlar.NewKeyValues(dmd, kevlar.JsonExt)
	if err != nil {
		return da.EndWithError(err)
	}

	rdp, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return da.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(rdp, data.SetupProperties)
	if err != nil {
		return da.EndWithError(err)
	}

	for _, id := range ids {

		if err = getProductDownloads(id, rdx, kvdm); err != nil {
			return da.EndWithError(err)
		}

	}

	return nil
}

func getProductDownloads(id string, rdx kevlar.ReadableRedux, kv kevlar.KeyValues) error {
	return nil
}
