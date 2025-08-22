package cli

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

func RemoveDownloadsHandler(u *url.URL) error {

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

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return RemoveDownloads(id, ii, rdx)
}

func RemoveDownloads(id string, ii *InstallInfo, rdx redux.Writeable) error {

	rda := nod.NewProgress("removing downloads...")
	defer rda.Done()

	printInstallInfoParams(ii, true, id)

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return err
	}

	productDetails, err := getProductDetails(id, rdx, ii.force)
	if err != nil {
		return err
	}

	if err = removeProductDownloadLinks(id, productDetails, ii, downloadsDir); err != nil {
		return err
	}

	rda.Increment()

	return nil
}

func removeProductDownloadLinks(id string,
	productDetails *vangogh_integration.ProductDetails,
	ii *InstallInfo,
	downloadsDir string) error {

	rdla := nod.Begin(" removing downloads for %s...", productDetails.Title)
	defer rdla.Done()

	idPath := filepath.Join(downloadsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rdla.EndWithResult("product downloads dir not present")
		return nil
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		rdla.EndWithResult("no links are matching operating params")
		return nil
	}

	for _, dl := range dls {

		// if we don't do this - product downloads dir itself will be removed
		if dl.LocalFilename == "" {
			continue
		}

		path := filepath.Join(downloadsDir, id, dl.LocalFilename)

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fa.EndWithResult("not present")
			continue
		}

		if err := os.Remove(path); err != nil {
			return err
		}

		fa.Done()
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)
	if entries, err := os.ReadDir(productDownloadsDir); err == nil && len(entries) == 0 {
		rdda := nod.Begin(" removing empty product downloads directory...")
		if err = os.Remove(productDownloadsDir); err != nil {
			return err
		}
		rdda.Done()
	} else {
		return err
	}

	return nil
}
