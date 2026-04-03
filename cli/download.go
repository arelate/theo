package cli

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arelate/southern_light/egs_integration"
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
		Origin:          data.VangoghOrigin,
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
		DownloadTypes:   downloadTypes,
		force:           q.Has("force"),
	}

	if q.Has("steam") {
		ii.Origin = data.SteamOrigin
	}

	if q.Has("epic-games") {
		ii.Origin = data.EpicGamesOrigin
	}

	var manualUrlFilter []string
	if q.Has("manual-url-filter") {
		manualUrlFilter = strings.Split(q.Get("manual-url-filter"), ",")
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

func vangoghDownloadData(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable, manualUrlFilter ...string) error {

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	if err := originHasFreeSpace(id, downloadsDir, ii, originData, manualUrlFilter); err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if token, ok := rdx.GetLastVal(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty); ok && token != "" {
		dc.SetAuthorizationBearer(token)
	}

	dls := originData.ProductDetails.DownloadLinks.
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

func steamDownloadData(steamAppId string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {
	steamAppsDir := data.Pwd.AbsDirPath(data.SteamApps)

	if err := originHasFreeSpace(steamAppId, steamAppsDir, ii, originData); err != nil {
		return err
	}

	return steamUpdateApp(steamAppId, ii.OperatingSystem, rdx)
}

func egsDownloadChunks(appName string, ii *InstallInfo, originData *data.OriginData) error {

	edca := nod.NewProgress("downloading EGS chunks...")
	edca.Done()

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	if err := originHasFreeSpace(appName, downloadsDir, ii, originData); err != nil {
		return err
	}

	edca.TotalInt(len(originData.Manifest.ChunkList.Chunks))

	cdnUrls, err := originData.GameManifest.Urls()
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	var cdnUrl *url.URL
	for _, cu := range cdnUrls {
		cdnUrl = cu
		break
	}

	if cdnUrl == nil {
		return errors.New("downloading EGS chunks requires CDN url")
	}

	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, ii.OperatingSystem)

	originalPath := strings.TrimSuffix(cdnUrl.Path, filepath.Base(cdnUrl.Path))
	cdnUrl.RawQuery = ""

	for _, chunk := range originData.Manifest.ChunkList.Chunks {

		chunkPath := chunk.Path(originData.Manifest.Metadata.FeatureLevel)
		cdnUrl.Path = path.Join(originalPath, chunkPath)

		if err = dc.Download(cdnUrl, ii.force, nil, absChunksDownloadsDir, chunkPath); err != nil {
			return err
		}

		edca.Increment()
	}

	return nil
}
