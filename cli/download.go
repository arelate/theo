package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
)

func DownloadHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return Download(operatingSystems, langCodes, downloadTypes, force, ids...)
}

func Download(operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	force bool,
	ids ...string) error {

	da := nod.NewProgress("downloading game data from vangogh...")
	defer da.EndWithResult("done")

	vangogh_integration.PrintParams(ids, operatingSystems, langCodes, downloadTypes, true)

	da.TotalInt(len(ids))

	for _, id := range ids {

		metadata, err := getTheoMetadata(id, force)
		if err != nil {
			return da.EndWithError(err)
		}

		if err = downloadProductFiles(id, metadata, operatingSystems, langCodes, downloadTypes, force); err != nil {
			return da.EndWithError(err)
		}

		da.Increment()
	}

	return nil
}

func downloadProductFiles(id string,
	metadata *vangogh_integration.TheoMetadata,
	operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	force bool) error {

	gpdla := nod.Begin(" downloading %s...", metadata.Title)
	defer gpdla.End()

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return gpdla.EndWithError(err)
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return gpdla.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties)
	if err != nil {
		return gpdla.EndWithError(err)
	}

	dc := dolo.DefaultClient

	if username, ok := rdx.GetLastVal(data.SetupProperties, data.VangoghUsernameProperty); ok && username != "" {
		if password, sure := rdx.GetLastVal(data.SetupProperties, data.VangoghPasswordProperty); sure && password != "" {
			dc.SetBasicAuth(username, password)
		}
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(operatingSystems...).
		FilterLanguageCodes(langCodes...).
		FilterDownloadTypes(downloadTypes...)

	if len(dls) == 0 {
		return gpdla.EndWithError(errors.New("no links are matching operating params"))
	}

	for _, dl := range dls {

		vr := vangogh_integration.ParseValidationResult(dl.ValidationResult)
		if vr != vangogh_integration.ValidatedSuccessfully &&
			vr != vangogh_integration.ValidatedMissingChecksum {
			continue
		}

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
