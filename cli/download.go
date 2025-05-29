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
	"slices"
	"strings"
)

func DownloadHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)

	q := u.Query()

	var manualUrlFilter []string
	if q.Has("manual-url-filter") {
		manualUrlFilter = strings.Split(q.Get("manual-url-filter"), ",")
	}

	force := q.Has("force")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return Download(operatingSystems, langCodes, downloadTypes, manualUrlFilter, rdx, force, ids...)
}

func Download(operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	manualUrlFilter []string,
	rdx redux.Writeable,
	force bool,
	ids ...string) error {

	da := nod.NewProgress("downloading game data from the server...")
	defer da.Done()

	vangogh_integration.PrintParams(ids, operatingSystems, langCodes, downloadTypes, true)

	da.TotalInt(len(ids))

	for _, id := range ids {

		productDetails, err := GetProductDetails(id, rdx, force)
		if err != nil {
			return err
		}

		if err = downloadProductFiles(id, productDetails, operatingSystems, langCodes, downloadTypes, manualUrlFilter, rdx, force); err != nil {
			return err
		}

		da.Increment()
	}

	return nil
}

func downloadProductFiles(id string,
	productDetails *vangogh_integration.ProductDetails,
	operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	manualUrlFilter []string,
	rdx redux.Readable,
	force bool) error {

	gpdla := nod.Begin(" downloading %s...", productDetails.Title)
	defer gpdla.Done()

	if err := rdx.MustHave(data.ServerConnectionProperties); err != nil {
		return err
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	if err = hasFreeSpaceForProduct(productDetails, downloadsDir,
		operatingSystems, langCodes, downloadTypes, force); err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if username, ok := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerUsernameProperty); ok && username != "" {
		if password, sure := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerPasswordProperty); sure && password != "" {
			dc.SetBasicAuth(username, password)
		}
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(operatingSystems...).
		FilterLanguageCodes(langCodes...).
		FilterDownloadTypes(downloadTypes...)

	if len(dls) == 0 {
		return errors.New("no links are matching operating params")
	}

	for _, dl := range dls {

		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}

		if dl.ValidationResult != vangogh_integration.ValidatedSuccessfully &&
			dl.ValidationResult != vangogh_integration.ValidatedMissingChecksum {
			errMsg := fmt.Sprintf("%s validation status %s prevented download", dl.Name, dl.ValidationResult)
			nod.LogError(errors.New(errMsg))
			continue
		}

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		fileUrl, err := data.ServerUrl(rdx,
			data.ServerFilesPath, map[string]string{
				"manual-url":    dl.ManualUrl,
				"id":            id,
				"download-type": dl.Type.String(),
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
