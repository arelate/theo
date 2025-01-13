package cli

import (
	"crypto/md5"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"golang.org/x/exp/slices"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)

	return Validate(operatingSystems, langCodes, downloadTypes, ids...)
}

func Validate(operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	ids ...string) error {

	va := nod.NewProgress("validating downloads...")
	defer va.EndWithResult("done")

	vangogh_integration.PrintParams(ids, operatingSystems, langCodes, downloadTypes, true)

	for _, id := range ids {

		metadata, err := getTheoMetadata(id, false)
		if err != nil {
			return va.EndWithError(err)
		}

		if err = validateLinks(id, operatingSystems, langCodes, downloadTypes, metadata); err != nil {
			return va.EndWithError(err)
		}
	}

	return nil
}

func validateLinks(id string,
	operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	metadata *vangogh_integration.TheoMetadata) error {

	vla := nod.NewProgress("validating %s...", metadata.Title)
	defer vla.End()

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return vla.EndWithError(err)
	}

	dls := metadata.DownloadLinks.
		FilterOperatingSystems(operatingSystems...).
		FilterLanguageCodes(langCodes...).
		FilterDownloadTypes(downloadTypes...)

	if len(dls) == 0 {
		vla.EndWithResult("no links are matching operating params")
		return nil
	}

	vla.TotalInt(len(dls))

	results := make([]ValidationResult, 0, len(dls))

	for _, dl := range dls {
		vr, err := validateLink(id, dl, downloadsDir)
		if err != nil {
			vla.Error(err)
		}
		results = append(results, vr)
	}

	vla.EndWithResult(summarizeValidationResults(results))

	return nil
}

func validateLink(id string, dl vangogh_integration.TheoDownloadLink, downloadsDir string) (ValidationResult, error) {

	dla := nod.NewProgress(" - %s...", dl.LocalFilename)
	defer dla.End()

	absDownloadPath := filepath.Join(downloadsDir, id, dl.LocalFilename)

	var stat os.FileInfo
	var err error

	if stat, err = os.Stat(absDownloadPath); os.IsNotExist(err) {
		dla.EndWithResult(ValResFileNotFound)
		return ValResFileNotFound, nil
	}

	if dl.Md5 == "" {
		dla.EndWithResult(ValResMissingChecksum)
		return ValResMissingChecksum, nil
	}

	dla.Total(uint64(stat.Size()))

	localFile, err := os.Open(absDownloadPath)
	if err != nil {
		return ValResError, dla.EndWithError(err)
	}

	h := md5.New()
	if err = dolo.CopyWithProgress(h, localFile, dla); err != nil {
		return ValResError, dla.EndWithError(err)
	}

	computedMd5 := fmt.Sprintf("%x", h.Sum(nil))
	if dl.Md5 == computedMd5 {
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
