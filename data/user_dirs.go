package data

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"os"
	"path/filepath"
)

func UserDataHomeDir() (string, error) {
	switch CurrentOs() {
	case vangogh_integration.Linux:
		uhd, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(uhd, ".local", "share"), nil
	case vangogh_integration.Windows:
		// TODO: verify that Windows user data home is also os.UserConfigDir
		fallthrough
	case vangogh_integration.MacOS:
		return os.UserConfigDir()
	default:
		panic("unsupported operating system")
	}
}
