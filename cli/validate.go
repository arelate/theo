package cli

import (
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
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

type ValidationResult string

const (
	ValResMismatch        = "mismatch"
	ValResError           = "error"
	ValResMissingChecksum = "missing checksum"
	ValResFileNotFound    = "file not found"
	ValResValid           = "valid"
)

var allValidationResults = []ValidationResult{
	ValResMismatch,
	ValResError,
	ValResMissingChecksum,
	ValResFileNotFound,
	ValResValid,
}

var valResMessageTemplates = map[ValidationResult]string{
	ValResMismatch:        "%s files did not match expected checksum",
	ValResError:           "%s files encountered errors during validation",
	ValResMissingChecksum: "%s files are missing checksums",
	ValResFileNotFound:    "%s files were not found",
	ValResValid:           "%s files are matching checksums",
}

func ValidateHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	os := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		os = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
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
		OperatingSystem: os,
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

	return Validate(id, ii, manualUrlFilter...)
}

func Validate(id string,
	ii *InstallInfo,
	manualUrlFilter ...string) error {

	va := nod.Begin("validating %s: %s...", ii.Origin, id)
	defer va.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	originData, err := originGetData(id, ii, rdx, false)
	if err != nil {
		return err
	}

	switch ii.Origin {
	case data.VangoghOrigin:
		return vangoghValidateData(id, ii, originData, rdx, manualUrlFilter...)
	case data.SteamOrigin:
		return steamValidateData(id, ii, rdx)
	case data.EpicGamesOrigin:
		return egsValidateChunks(id, ii, originData)
	default:
		return ii.Origin.ErrUnsupportedOrigin()
	}
}

func steamValidateData(steamAppId string, ii *InstallInfo, rdx redux.Readable) error {
	return steamUpdateApp(steamAppId, ii.OperatingSystem, rdx)
}

func vangoghValidateData(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable, manualUrlFilter ...string) error {
	va := nod.NewProgress("validating downloads...")
	defer va.Done()

	manualUrlChecksums, err := getManualUrlChecksums(id, rdx, ii.force)
	if err != nil {
		return err
	}

	var mismatchedManualUrls []string
	if mismatchedManualUrls, err = vangoghValidateLinks(id, ii, manualUrlFilter, originData.ProductDetails, manualUrlChecksums); err != nil {
		return err
	} else if len(mismatchedManualUrls) > 0 {

		// redownload and revalidate any manual-urls that resulted in mismatched checksums

		ii.force = true

		if err = Download(id, ii, nil, mismatchedManualUrls...); err != nil {
			return err
		}

		if _, err = vangoghValidateLinks(id, ii, manualUrlFilter, originData.ProductDetails, manualUrlChecksums); err != nil {
			return err
		}
	}

	return nil
}

func vangoghValidateLinks(id string,
	ii *InstallInfo,
	manualUrlFilter []string,
	productDetails *vangogh_integration.ProductDetails,
	manualUrlChecksums map[string]string) ([]string, error) {

	vla := nod.NewProgress("validating %s...", productDetails.Title)
	defer vla.Done()

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		return nil, errors.New("no links are matching operating params")
	}

	vla.TotalInt(len(dls))

	results := make([]ValidationResult, 0, len(dls))

	var mismatchedManualUrls []string

	for _, dl := range dls {
		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}

		vr, err := vangoghValidateLink(id, &dl, manualUrlChecksums[dl.ManualUrl], downloadsDir)
		if err != nil {
			vla.Error(err)
		}

		if vr == ValResMismatch {
			mismatchedManualUrls = append(mismatchedManualUrls, dl.ManualUrl)
		}

		results = append(results, vr)
	}

	vla.EndWithResult(summarizeValidationResults(results))

	return mismatchedManualUrls, nil
}

func vangoghValidateLink(id string, link *vangogh_integration.ProductDownloadLink, manualUrlMd5 string, downloadsDir string) (ValidationResult, error) {

	dla := nod.NewProgress(" - %s...", link.LocalFilename)
	defer dla.Done()

	absDownloadPath := filepath.Join(downloadsDir, id, link.LocalFilename)

	var stat os.FileInfo
	var err error

	if stat, err = os.Stat(absDownloadPath); os.IsNotExist(err) {
		dla.EndWithResult(ValResFileNotFound)
		return ValResFileNotFound, nil
	}

	if manualUrlMd5 == "" {
		dla.EndWithResult(ValResMissingChecksum)
		return ValResMissingChecksum, nil
	}

	dla.Total(uint64(stat.Size()))

	localFile, err := os.Open(absDownloadPath)
	if err != nil {
		return ValResError, err
	}

	h := md5.New()
	if err = dolo.CopyWithProgress(h, localFile, dla); err != nil {
		return ValResError, err
	}

	computedMd5 := fmt.Sprintf("%x", h.Sum(nil))
	if manualUrlMd5 == computedMd5 {
		dla.EndWithResult(ValResValid)
		return ValResValid, nil
	} else {
		dla.EndWithResult(ValResMismatch)
		return ValResMismatch, nil
	}
}

func summarizeValidationResults(results []ValidationResult) string {

	desc := make([]string, 0)

	for _, vr := range allValidationResults {
		if slices.Contains(results, vr) {
			someAll := "some"
			if isSameResult(vr, results) {
				someAll = "all"
			}
			desc = append(desc, fmt.Sprintf(valResMessageTemplates[vr], someAll))
		}
	}

	return strings.Join(desc, "; ")
}

func isSameResult(exp ValidationResult, results []ValidationResult) bool {
	for _, vr := range results {
		if vr != exp {
			return false
		}
	}
	return true
}

func egsValidateChunks(appName string, ii *InstallInfo, originData *data.OriginData) error {

	evca := nod.NewProgress("validating EGS chunks for %s-%s...", appName, ii.OperatingSystem)
	defer evca.Done()

	evca.Total(uint64(egsManifestSize(originData.Manifest)))

	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, ii.OperatingSystem)

	for _, chunk := range originData.Manifest.ChunkList.Chunks {

		chunkPath := chunk.Path(originData.Manifest.Metadata.FeatureLevel)

		absChunkFilename := filepath.Join(absChunksDownloadsDir, chunkPath)

		chunkFile, err := os.Open(absChunkFilename)
		if err != nil {
			return err
		}

		chunkReader, err := egs_integration.ReadChunk(chunkFile)
		if err != nil {
			return err
		}

		shaSum := sha1.New()

		if _, err = io.Copy(shaSum, chunkReader); err != nil {
			return err
		}

		expectedShaSum := fmt.Sprintf("%x", chunk.ShaHash)
		actualShaSum := fmt.Sprintf("%x", shaSum.Sum(nil))

		if expectedShaSum != actualShaSum {
			return errors.New("failed validation for " + chunkPath)
		}

		evca.Progress(chunk.FileSize)
	}

	evca.EndWithResult("valid")

	return nil
}
