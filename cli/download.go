package cli

import (
	"errors"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

func DownloadHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.ServerConnectionProperties, vangogh_integration.TitleProperty, vangogh_integration.SlugProperty)
	if err != nil {
		return err
	}

	return Download(operatingSystems, langCodes, downloadTypes, rdx, force, ids...)
}

func Download(operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	rdx redux.Writeable,
	force bool,
	ids ...string) error {

	da := nod.NewProgress("downloading game data from the server...")
	defer da.Done()

	vangogh_integration.PrintParams(ids, operatingSystems, langCodes, downloadTypes, true)

	da.TotalInt(len(ids))

	for _, id := range ids {

		metadata, err := getTheoMetadata(id, rdx, force)
		if err != nil {
			return err
		}

		if err = downloadProductFiles(id, metadata, operatingSystems, langCodes, downloadTypes, rdx, force); err != nil {
			return err
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
	rdx redux.Readable,
	force bool) error {

	gpdla := nod.Begin(" downloading %s...", metadata.Title)
	defer gpdla.Done()

	if err := rdx.MustHave(data.ServerConnectionProperties); err != nil {
		return err
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if username, ok := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerUsernameProperty); ok && username != "" {
		if password, sure := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerPasswordProperty); sure && password != "" {
			dc.SetBasicAuth(username, password)
		}
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(operatingSystems...).
		FilterLanguageCodes(langCodes...).
		FilterDownloadTypes(downloadTypes...)

	if len(dls) == 0 {
		return errors.New("no links are matching operating params")
	}

	for _, dl := range dls {

		vr := vangogh_integration.ParseValidationResult(dl.ValidationResult)
		if vr != vangogh_integration.ValidatedSuccessfully &&
			vr != vangogh_integration.ValidatedMissingChecksum &&
			vr != vangogh_integration.ValidatedWithGeneratedChecksum {
			errMsg := fmt.Sprintf("%s validation status %s prevented download", dl.Name, dl.ValidationResult)
			nod.LogError(errors.New(errMsg))
			continue
		}

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		fileUrl, err := data.ServerUrl(rdx,
			data.ServerFilesPath, map[string]string{
				"manual-url": dl.ManualUrl,
			})
		if err != nil {
			fa.EndWithResult(err.Error())
			continue
		}

		if err = dc.Download(fileUrl, force, fa, downloadsDir, id, dl.LocalFilename); err != nil {
			fa.EndWithResult(err.Error())
			continue
		}

		fa.Done()
	}

	return nil
}
