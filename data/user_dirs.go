package data

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"os"
	"path/filepath"
)

func UserDataHomeDir() (string, error) {
	switch CurrentOS() {
	case vangogh_integration.Linux:
		uhd, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(uhd, ".local", "share"), nil
	case vangogh_integration.Windows:
		fallthrough
	case vangogh_integration.MacOS:
		return os.UserConfigDir()
	default:
		panic("unsupported operating system")
	}
}
