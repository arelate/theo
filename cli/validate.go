package cli

import (
	"crypto/md5"
	"fmt"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
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
	operatingSystems, langCodes, _ := OsLangCodeDownloadType(u)

	return Validate(ids, operatingSystems, langCodes)
}

func Validate(ids []string, operatingSystems []vangogh_local_data.OperatingSystem, langCodes []string) error {

	va := nod.NewProgress("validating downloads...")
	defer va.End()

	vangogh_local_data.PrintParams(ids, operatingSystems, langCodes, nil, true)

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return va.EndWithError(err)
	}

	for _, id := range ids {

		if metadata, err := GetDownloadMetadata(id, operatingSystems, langCodes, nil, false); err == nil {
			if err = validateLinks(id, metadata, downloadsDir); err != nil {
				return va.EndWithError(err)
			}
		} else {
			return va.EndWithError(err)
		}
	}

	va.EndWithResult("done")

	return nil
}

func validateLinks(id string, metadata *vangogh_local_data.DownloadMetadata, downloadsDir string) error {

	vla := nod.NewProgress("validating %s...", metadata.Title)
	defer vla.End()

	vla.TotalInt(len(metadata.DownloadLinks))

	results := make([]ValidationResult, 0, len(metadata.DownloadLinks))

	for _, dl := range metadata.DownloadLinks {
		vr, err := validateLink(id, dl, downloadsDir)
		if err != nil {
			vla.Error(err)
		}
		results = append(results, vr)
	}

	vla.EndWithResult(summarizeValidationResults(results))

	return nil
}

func validateLink(id string, dl vangogh_local_data.DownloadLink, downloadsDir string) (ValidationResult, error) {

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
