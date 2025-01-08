package data

import (
	"errors"
	"github.com/arelate/vangogh_local_data"
	"os"
	"os/exec"
)

func InstalledTheoOrCurrentProcessPath() (string, error) {

	binFilename := "theo"

	switch CurrentOS() {
	case vangogh_local_data.Windows:
		binFilename += ".exe"
	case vangogh_local_data.Linux:
		fallthrough
	case vangogh_local_data.MacOS:
	// do nothing
	default:
		return "", errors.New("unsupported operating system")
	}

	if binPath, err := exec.LookPath(binFilename); err == nil && binPath != "" {
		return binPath, nil
	} else if len(os.Args) > 0 {
		return os.Args[0], nil
	}

	return "", errors.New("theo binary not found, please add it to a PATH location")
}
