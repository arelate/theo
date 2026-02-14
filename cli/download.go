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

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
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

	vangogh_integration.PrintParams([]string{id},
		[]vangogh_integration.OperatingSystem{ii.OperatingSystem},
		[]string{ii.LangCode},
		ii.DownloadTypes,
		true)

	// always get the latest product details for download purposes
	productDetails, err := getProductDetails(id, rdx, true)
	if err != nil {
		return err
	}

	if err = resolveInstallInfo(id, ii, productDetails, rdx, currentOsThenWindows); err != nil {
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

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	if err := hasFreeSpaceForProduct(productDetails, downloadsDir, ii, manualUrlFilter); err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if token, ok := rdx.GetLastVal(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty); ok && token != "" {
		dc.SetAuthorizationBearer(token)
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		return errors.New("no links are matching operating params")
	}

	for _, dl := range dls {

		if dl.LocalFilename == "" {
			return errors.New("unresolved local filename for manual-url " + dl.ManualUrl)
		}

		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}

		if dl.ValidationStatus != vangogh_integration.ValidationStatusSuccess &&
			dl.ValidationStatus != vangogh_integration.ValidationStatusSelfValidated &&
			dl.ValidationStatus != vangogh_integration.ValidationStatusMissingChecksum {
			errMsg := fmt.Sprintf("%s validation status %s prevented download", dl.Name, dl.ValidationStatus)
			return errors.New(errMsg)
		}

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		query := url.Values{
			"manual-url":    {dl.ManualUrl},
			"id":            {id},
			"download-type": {dl.DownloadType.String()},
		}

		fileUrl, err := data.VangoghUrl(data.HttpFilesPath, query, rdx)
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
