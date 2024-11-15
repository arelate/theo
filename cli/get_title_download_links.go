package cli

import (
	"encoding/json"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"golang.org/x/exp/slices"
)

func GetTitleDownloadLinks(id string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	kv kevlar.KeyValues,
	force bool) (string, []vangogh_local_data.DownloadLink, error) {

	// if not download-type was specified - add all to avoid filtering on them
	if downloadTypes == nil {
		downloadTypes = []vangogh_local_data.DownloadType{
			vangogh_local_data.Installer,
			vangogh_local_data.DLC,
		}
	}

	fdla := nod.Begin("filtering downloads metadata links for %s...", id)
	defer fdla.End()

	if has, err := kv.Has(id); err == nil {
		if !has {
			if err = GetDownloadsMetadata([]string{id}, force); err != nil {
				return "", nil, fdla.EndWithError(err)
			}
		}
	} else {
		return "", nil, fdla.EndWithError(err)
	}

	dmrc, err := kv.Get(id)
	if err != nil {
		return "", nil, fdla.EndWithError(err)
	}
	defer dmrc.Close()

	var downloadMetadata vangogh_local_data.DownloadMetadata
	if err = json.NewDecoder(dmrc).Decode(&downloadMetadata); err != nil {
		return "", nil, fdla.EndWithError(err)
	}

	var downloadLinks []vangogh_local_data.DownloadLink
	for _, link := range downloadMetadata.DownloadLinks {
		linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
		linkType := vangogh_local_data.ParseDownloadType(link.Type)
		linkLangCode := link.LanguageCode

		if slices.Contains(operatingSystems, linkOs) &&
			slices.Contains(langCodes, linkLangCode) &&
			slices.Contains(downloadTypes, linkType) {
			downloadLinks = append(downloadLinks, link)
		}
	}

	if len(downloadLinks) == 0 {
		fdla.EndWithResult("no links found (%d total)", len(downloadMetadata.DownloadLinks))
		return downloadMetadata.Title, nil, nil
	}

	return downloadMetadata.Title, downloadLinks, nil
}
