package cli

import (
	"errors"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"net/url"
	"slices"
	"strconv"
)

const preserveFreeSpacePercent = 1

func HasFreeSpaceHandler(u *url.URL) error {

	q := u.Query()

	bs := q.Get("bytes")

	var bytes int64
	if bi, err := strconv.ParseInt(bs, 10, 64); err == nil {
		bytes = bi
	} else {
		return err
	}

	path := q.Get("path")
	if path == "" {
		path = "/"
	}

	if _, err := HasFreeSpace(path, bytes); err != nil {
		return err
	}

	return nil
}

func HasFreeSpace(path string, bytes int64) (bool, error) {

	var relPath string
	if userHomeDataRel, err := data.RelToUserDataHome(path); err == nil {
		relPath = userHomeDataRel
	} else {
		return false, err
	}

	hfsa := nod.Begin("checking free space at %s...", relPath)
	defer hfsa.Done()

	currentOs := data.CurrentOs()

	var availableBytes int64
	var err error

	switch currentOs {
	case vangogh_integration.Windows:
		availableBytes, err = windowsFreeSpace(path)
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		availableBytes, err = nixFreeSpace(path)
	default:
		return false, errors.New("unsupported operating system")
	}

	if err != nil {
		return false, err
	}

	// we don't want to consume all available space, so reserving
	// specified percentage of available capacity before the checks
	availableBytes = (100 - preserveFreeSpacePercent) * availableBytes / 100

	switch availableBytes > bytes {
	case true:
		hfsa.EndWithResult("enough for %s (%s free)",
			vangogh_integration.FormatBytes(bytes),
			vangogh_integration.FormatBytes(availableBytes))
		return true, nil
	case false:
		hfsa.EndWithResult("not enough for %s (%s free)",
			vangogh_integration.FormatBytes(bytes),
			vangogh_integration.FormatBytes(availableBytes))
		return false, nil
	}

	return availableBytes > bytes, nil
}

func hasFreeSpaceForProduct(
	productDetails *vangogh_integration.ProductDetails,
	targetPath string,
	operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	manualUrlFilter []string,
	force bool) error {

	var totalEstimatedBytes int64

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(operatingSystems...).
		FilterLanguageCodes(langCodes...).
		FilterDownloadTypes(downloadTypes...)

	for _, dl := range dls {
		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}
		totalEstimatedBytes += dl.EstimatedBytes
	}

	if ok, err := HasFreeSpace(targetPath, totalEstimatedBytes); err != nil {
		return err
	} else if !ok && !force {
		return fmt.Errorf("not enough space for %s at %s"+productDetails.Id, targetPath)
	} else {
		return nil
	}
}
