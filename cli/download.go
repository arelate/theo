package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

const defaultLangCode = "en"

func DownloadHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return Download(ids, operatingSystems, langCodes, downloadTypes, force)
}

func Download(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	da := nod.NewProgress("downloading game data from vangogh...")
	defer da.End()

	PrintParams(ids, operatingSystems, langCodes, downloadTypes)

	da.TotalInt(len(ids))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return da.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties)
	if err != nil {
		return da.EndWithError(err)
	}

	for _, id := range ids {

		if title, links, err := GetTitleDownloadLinks(id,
			operatingSystems,
			langCodes,
			downloadTypes,
			force); err == nil {
			if err = getProductDownloadLinks(id,
				title,
				links,
				rdx,
				force); err != nil {
				return da.EndWithError(err)
			}
		} else {
			return da.EndWithError(err)
		}

		da.Increment()
	}

	da.EndWithResult("done")

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

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return gpdla.EndWithError(err)
	}

	for _, dl := range downloadLinks {

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		fileUrl, err := data.VangoghUrl(rdx,
			data.VangoghFilesPath, map[string]string{
				"manual-url": dl.ManualUrl,
			})
		if err != nil {
			_ = fa.EndWithError(err)
			continue
		}

		if err := dc.Download(fileUrl, force, fa, downloadsDir, id, dl.LocalFilename); err != nil {
			_ = fa.EndWithError(err)
			continue
		}

		fa.EndWithResult("done")
	}

	return nil
}
