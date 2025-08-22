package cli

import (
	"fmt"
	"slices"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
)

const preserveFreeSpacePercent = 1

func hasFreeSpaceForProduct(
	productDetails *vangogh_integration.ProductDetails,
	targetPath string,
	ii *InstallInfo,
	manualUrlFilter []string) error {

	var totalEstimatedBytes int64

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	for _, dl := range dls {
		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}
		totalEstimatedBytes += dl.EstimatedBytes
	}

	if ok, err := hasFreeSpaceForBytes(targetPath, totalEstimatedBytes); err != nil {
		return err
	} else if !ok && !ii.force {
		return fmt.Errorf("not enough space for %s at %s", productDetails.Id, targetPath)
	} else {
		return nil
	}
}

func hasFreeSpaceForBytes(path string, bytes int64) (bool, error) {

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
		return false, currentOs.ErrUnsupported()
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
