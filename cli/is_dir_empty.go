package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"os"
)

func osIsDirEmpty(path string) (bool, error) {
	if entries, err := os.ReadDir(path); err == nil && len(entries) == 0 {
		return true, nil
	} else if err == nil {
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			return macOsIsDirEmptyOrDsStoreOnly(entries), nil
		case vangogh_integration.Linux:
			// currently not tracking any special cases for Linux
			return false, nil
		case vangogh_integration.Windows:
			// currently not tracking any special cases for Windows
			return false, nil
		default:
			return false, currentOs.ErrUnsupported()
		}
	} else {
		return false, err
	}
}
