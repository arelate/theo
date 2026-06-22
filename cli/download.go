package cli

import (
	"net/url"
	"strings"

	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func DownloadHandler(u *url.URL) error {

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
		Origin:          data.VangoghOrigin,
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
		DownloadTypes:   downloadTypes,
		force:           q.Has(vangogh_integration.UrlForceParameter),
	}

	if q.Has(vangogh_integration.UrlSteamParameter) {
		ii.Origin = data.SteamOrigin
	}

	if q.Has(vangogh_integration.UrlEpicGamesParameter) {
		ii.Origin = data.EpicGamesOrigin
	}

	var manualUrlFilter []string
	if q.Has(vangogh_integration.UrlManualUrlFilterParameter) {
		manualUrlFilter = strings.Split(q.Get(vangogh_integration.UrlManualUrlFilterParameter), ",")
	}

	return Download(id, ii, nil, manualUrlFilter...)
}

func Download(id string,
	ii *InstallInfo,
	originData *data.OriginData,
	manualUrlFilter ...string) error {

	da := nod.Begin("downloading product data...")
	defer da.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	vangogh_integration.PrintParams([]string{id},
		[]vangogh_integration.OperatingSystem{ii.OperatingSystem},
		[]string{ii.LangCode},
		ii.DownloadTypes,
		true)

	if originData == nil {
		originData, err = originGetData(id, ii, rdx, true)
		if err != nil {
			return err
		}
	}

	if err = originDownloadData(id, ii, originData, manualUrlFilter, rdx); err != nil {
		return err
	}

	return nil
}

func originGetData(id string, ii *InstallInfo, rdx redux.Writeable, force bool) (*data.OriginData, error) {

	originData := new(data.OriginData)
	var err error

	switch ii.Origin {
	case data.VangoghOrigin:
		if originData.ProductDetails, err = vangoghGetProductDetails(id, rdx, force); err != nil {
			return nil, err
		}
	case data.SteamOrigin:
		if originData.AppInfoKv, err = steamGetAppInfoKv(id, rdx, force); err != nil {
			return nil, err
		}
	case data.EpicGamesOrigin:

		var gameAssetsOs []vangogh_integration.OperatingSystem
		gameAssetsOs, err = egsGameAssetOperatingSystems(id, ii.force)
		if err != nil {
			return nil, err
		}

		setInstallInfoDefaults(ii, gameAssetsOs)

		var gameAsset *egs_integration.GameAsset
		if gameAsset, err = egsGetGameAsset(id, ii); err != nil {
			return nil, err
		}
		if originData.CatalogItem, err = egsGetCatalogItem(gameAsset, ii, rdx); err != nil {
			return nil, err
		}

		// the data items below must be the latest version from the origin when downloading, don't remove force parameter
		if originData.GameManifest, err = egsGetGameManifest(gameAsset, ii, force); err != nil {
			return nil, err
		}
		if originData.Manifest, err = egsGetManifest(gameAsset.AppName, originData.GameManifest, ii.OperatingSystem, force); err != nil {
			return nil, err
		}

	default:
		return nil, ii.Origin.ErrUnsupportedOrigin()
	}

	if err = ii.reduceOriginData(id, originData); err != nil {
		return nil, err
	}

	return originData, nil
}

func originDownloadData(id string,
	ii *InstallInfo,
	originData *data.OriginData,
	manualUrlFilter []string,
	rdx redux.Readable) error {

	odda := nod.Begin(" downloading %s: %s...", ii.Origin, id)
	defer odda.Done()

	switch ii.Origin {
	case data.VangoghOrigin:
		return vangoghDownloadData(id, ii, originData, rdx, manualUrlFilter...)
	case data.SteamOrigin:
		return steamDownloadData(id, ii, originData, rdx)
	case data.EpicGamesOrigin:
		return egsDownloadChunks(id, ii, originData)
	default:
		return ii.Origin.ErrUnsupportedOrigin()
	}
}
