package cli

import (
	"encoding/json"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"golang.org/x/exp/slices"
)

func GetTitleDownloadLinks(id string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
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

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return "", nil, err
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir, data.TitleProperty)
	if err != nil {
		return "", nil, err
	}

	downloadsMetadataDir, err := pathways.GetAbsRelDir(data.DownloadsMetadata)
	if err != nil {
		return "", nil, err
	}

	kvDownloadsMetadata, err := kevlar.NewKeyValues(downloadsMetadataDir, kevlar.JsonExt)
	if err != nil {
		return "", nil, err
	}

	if has, err := kvDownloadsMetadata.Has(id); err == nil {
		if !has {
			if err = GetDownloadsMetadata([]string{id}, force); err != nil {
				return "", nil, fdla.EndWithError(err)
			}
		}
	} else {
		return "", nil, fdla.EndWithError(err)
	}

	dmrc, err := kvDownloadsMetadata.Get(id)
	if err != nil {
		return "", nil, fdla.EndWithError(err)
	}
	defer dmrc.Close()

	var downloadMetadata vangogh_local_data.DownloadMetadata
	if err = json.NewDecoder(dmrc).Decode(&downloadMetadata); err != nil {
		return "", nil, fdla.EndWithError(err)
	}

	if err := rdx.AddValues(data.TitleProperty, id, downloadMetadata.Title); err != nil {
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
