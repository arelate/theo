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

func LoadOrFetchTheoMetadata(id string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) (*vangogh_local_data.TheoMetadata, error) {

	// if no download-type was specified - add all to avoid filtering on them
	if downloadTypes == nil ||
		(len(downloadTypes) == 1 && downloadTypes[0] == vangogh_local_data.AnyDownloadType) {
		downloadTypes = []vangogh_local_data.DownloadType{
			vangogh_local_data.Installer,
			vangogh_local_data.DLC,
		}
	}
	if operatingSystems == nil ||
		(len(operatingSystems) == 1 && operatingSystems[0] == vangogh_local_data.AnyOperatingSystem) {
		operatingSystems = []vangogh_local_data.OperatingSystem{
			vangogh_local_data.Windows,
			vangogh_local_data.MacOS,
			vangogh_local_data.Linux,
		}
	}

	fdla := nod.Begin("filtering theo metadata links for %s...", id)
	defer fdla.End()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return nil, err
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir,
		data.TitleProperty,
		data.SlugProperty)
	if err != nil {
		return nil, err
	}

	theoMetadataDir, err := pathways.GetAbsRelDir(data.TheoMetadata)
	if err != nil {
		return nil, err
	}

	kvTheoMetadata, err := kevlar.NewKeyValues(theoMetadataDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if has, err := kvTheoMetadata.Has(id); err == nil {
		if !has || force {
			if err = GetTheoMetadata([]string{id}, force); err != nil {
				return nil, fdla.EndWithError(err)
			}
		}
	} else {
		return nil, fdla.EndWithError(err)
	}

	dmrc, err := kvTheoMetadata.Get(id)
	if err != nil {
		return nil, fdla.EndWithError(err)
	}
	defer dmrc.Close()

	var downloadMetadata vangogh_local_data.TheoMetadata
	if err = json.NewDecoder(dmrc).Decode(&downloadMetadata); err != nil {
		return nil, fdla.EndWithError(err)
	}

	if err := rdx.ReplaceValues(data.TitleProperty, id, downloadMetadata.Title); err != nil {
		return nil, fdla.EndWithError(err)
	}

	if err := rdx.ReplaceValues(data.SlugProperty, id, downloadMetadata.Slug); err != nil {
		return nil, fdla.EndWithError(err)
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

	downloadMetadata.DownloadLinks = downloadLinks

	if len(downloadLinks) == 0 {
		fdla.EndWithResult("no links found (%d total)", len(downloadMetadata.DownloadLinks))
		return nil, nil
	}

	return &downloadMetadata, nil
}
