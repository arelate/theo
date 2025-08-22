package cli

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

func DownloadHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	var langCode string
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	var downloadTypes []vangogh_integration.DownloadType
	if q.Has(vangogh_integration.DownloadTypeProperty) {
		dts := strings.Split(q.Get(vangogh_integration.DownloadTypeProperty), ",")
		downloadTypes = vangogh_integration.ParseManyDownloadTypes(dts)
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
		DownloadTypes:   downloadTypes,
		force:           q.Has("force"),
	}

	var manualUrlFilter []string
	if q.Has("manual-url-filter") {
		manualUrlFilter = strings.Split(q.Get("manual-url-filter"), ",")
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return Download(id, ii, manualUrlFilter, rdx)
}

func Download(id string,
	ii *InstallInfo,
	manualUrlFilter []string,
	rdx redux.Writeable) error {

	da := nod.NewProgress("downloading from the server...")
	defer da.Done()

	printInstallInfoParams(ii, true, id)

	productDetails, err := getProductDetails(id, rdx, ii.force)
	if err != nil {
		return err
	}

	if err = downloadProductFiles(id, productDetails, ii, manualUrlFilter, rdx); err != nil {
		return err
	}

	da.Increment()

	return nil
}

func downloadProductFiles(id string,
	productDetails *vangogh_integration.ProductDetails,
	ii *InstallInfo,
	manualUrlFilter []string,
	rdx redux.Readable) error {

	gpdla := nod.Begin(" downloading %s...", productDetails.Title)
	defer gpdla.Done()

	if err := rdx.MustHave(data.ServerConnectionProperties); err != nil {
		return err
	}

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	if err = hasFreeSpaceForProduct(productDetails, downloadsDir, ii, manualUrlFilter); err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if username, ok := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerUsernameProperty); ok && username != "" {
		if password, sure := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerPasswordProperty); sure && password != "" {
			dc.SetBasicAuth(username, password)
		}
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

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
			return errors.New(errMsg)
		}

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		fileUrl, err := data.ServerUrl(rdx,
			data.HttpFilesPath, map[string]string{
				"manual-url":    dl.ManualUrl,
				"id":            id,
				"download-type": dl.Type.String(),
			})
		if err != nil {
			fa.EndWithResult(err.Error())
			continue
		}

		if err = dc.Download(fileUrl, ii.force, fa, downloadsDir, id, dl.LocalFilename); err != nil {
			fa.EndWithResult(err.Error())
			continue
		}

		fa.Done()
	}

	return nil
}
