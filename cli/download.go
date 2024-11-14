package cli

import (
	"encoding/json"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"runtime"
	"slices"
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

	return Download(ids, operatingSystems, langCodes, downloadTypes, force)
}

func Download(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	da := nod.NewProgress("downloading game data from vangogh...")
	defer da.End()

	da.TotalInt(len(ids))

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

		if err = getProductDownloads(id, rdx, kvdm, operatingSystems, langCodes, downloadTypes, force); err != nil {
			return da.EndWithError(err)
		}

		da.Increment()
	}

	da.EndWithResult("done")

	return nil
}

func getProductDownloads(id string,
	rdx kevlar.ReadableRedux,
	kv kevlar.KeyValues,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	dma := nod.Begin(" loading downloads metadata for %s...", id)
	defer dma.End()

	if has, err := kv.Has(id); err == nil {
		if !has {
			if err = GetDownloadsMetadata([]string{id}, force); err != nil {
				return dma.EndWithError(err)
			}
		}
	} else {
		return dma.EndWithError(err)
	}

	dmrc, err := kv.Get(id)
	if err != nil {
		return dma.EndWithError(err)
	}
	defer dmrc.Close()

	var downloadMetadata vangogh_local_data.DownloadMetadata
	if err = json.NewDecoder(dmrc).Decode(&downloadMetadata); err != nil {
		return dma.EndWithError(err)
	}

	var downloadLinks []vangogh_local_data.DownloadLink
	for _, link := range downloadMetadata.DownloadLinks {
		linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
		linkType := vangogh_local_data.ParseDownloadType(link.Type)
		linkLangCode := link.LanguageCode

		if slices.Contains(operatingSystems, linkOs) &&
			slices.Contains(downloadTypes, linkType) &&
			slices.Contains(langCodes, linkLangCode) {
			downloadLinks = append(downloadLinks, link)
		}
	}

	if err = getProductDownloadLinks(id, downloadMetadata.Title, downloadLinks, rdx, force); err != nil {
		return dma.EndWithError(err)
	}

	dma.EndWithResult("done")

	return nil
}

func getProductDownloadLinks(id, title string,
	downloadLinks []vangogh_local_data.DownloadLink,
	rdx kevlar.ReadableRedux,
	force bool) error {

	gpdla := nod.Begin(" downloading %s...", title)
	defer gpdla.End()

	if err := rdx.MustHave(data.SetupProperties); err != nil {
		return gpdla.EndWithError(err)
	}

	dc := dolo.DefaultClient

	if username, ok := rdx.GetLastVal(data.SetupProperties, data.VangoghUsernameProperty); ok && username != "" {
		if password, sure := rdx.GetLastVal(data.SetupProperties, data.VangoghPasswordProperty); sure && password != "" {
			dc.SetBasicAuth(username, password)
		}
	}

	ddp, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return gpdla.EndWithError(err)
	}

	for _, dl := range downloadLinks {

		fa := nod.NewProgress(" - %s", dl.LocalFilename)

		fileUrl, err := data.VangoghUrl(rdx,
			data.VangoghFilesPath, map[string]string{
				"manual-url": dl.ManualUrl,
			})
		if err != nil {
			_ = fa.EndWithError(err)
			continue
		}

		if err := dc.Download(fileUrl, force, fa, ddp, id, dl.LocalFilename); err != nil {
			_ = fa.EndWithError(err)
			continue
		}

		fa.EndWithResult("done")
	}

	return nil
}
