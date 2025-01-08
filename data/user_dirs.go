package data

import (
	"github.com/arelate/vangogh_local_data"
	"os"
	"path/filepath"
)

func UserDataHomeDir() (string, error) {
	switch CurrentOS() {
	case vangogh_local_data.Linux:
		uhd, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(uhd, ".local", "share"), nil
	case vangogh_local_data.Windows:
		fallthrough
	case vangogh_local_data.MacOS:
		return os.UserConfigDir()
	default:
		panic("unsupported operating system")
	}
}
