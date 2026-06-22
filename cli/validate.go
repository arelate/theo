package cli

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
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

	id := q.Get(vangogh_integration.UrlIdParameter)

	os := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.UrlOperatingSystemParameter) {
		os = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.UrlOperatingSystemParameter))
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
		OperatingSystem: os,
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
		return steamUpdateApp(id, ii.OperatingSystem, rdx)
	case data.EpicGamesOrigin:
		return egsValidateChunks(id, ii, originData)
	default:
		return ii.Origin.ErrUnsupportedOrigin()
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
