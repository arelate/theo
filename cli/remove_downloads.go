package cli

import (
	"net/url"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func RemoveDownloadsHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.UrlIdParameter)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.UrlOperatingSystemParameter) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.UrlOperatingSystemParameter))
	}

	var langCode string
	if q.Has(vangogh_integration.UrlLanguageCodeParameter) {
		langCode = q.Get(vangogh_integration.UrlLanguageCodeParameter)
	}

	var downloadTypes []vangogh_integration.DownloadType
	if q.Has(vangogh_integration.UrlDownloadTypeParameter) {
		dts := strings.Split(q.Get(vangogh_integration.UrlDownloadTypeParameter), ",")
		downloadTypes = vangogh_integration.ParseManyDownloadTypes(dts)
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
		DownloadTypes:   downloadTypes,
		force:           q.Has(vangogh_integration.UrlForceParameter),
	}

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	return RemoveDownloads(id, ii, rdx)
}

func RemoveDownloads(id string, ii *InstallInfo, rdx redux.Writeable) error {

	rda := nod.Begin("removing downloads...")
	defer rda.Done()

	vangogh_integration.PrintParams([]string{id},
		[]vangogh_integration.OperatingSystem{ii.OperatingSystem},
		[]string{ii.LangCode},
		ii.DownloadTypes,
		true)

	originData, err := originGetData(id, ii, rdx, false)
	if err != nil {
		return err
	}

	if err = originRemoveDownloads(id, ii, originData, rdx); err != nil {
		return err
	}

	return nil
}

func originRemoveDownloads(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable) error {

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	switch ii.Origin {
	case data.VangoghOrigin:
		if err := vangoghRemoveProductDownloadLinks(id, originData.ProductDetails, ii, downloadsDir); err != nil {
			return err
		}
	case data.SteamOrigin:
	// do nothing
	case data.EpicGamesOrigin:
		if err := egsRemoveChunks(id, ii.OperatingSystem, originData); err != nil {
			return err
		}
	default:
		return ii.Origin.ErrUnsupportedOrigin()
	}

	return nil
}
