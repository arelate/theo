package data

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"os"
	"os/exec"
)

func TheoExecutable() (string, error) {

	binFilename := "theo"

	switch CurrentOs() {
	case vangogh_integration.Windows:
		binFilename += ".exe"
	case vangogh_integration.Linux:
		fallthrough
	case vangogh_integration.MacOS:
	// do nothing
	default:
		return "", errors.New("unsupported operating system")
	}

	// check PATH first and make sure the location specified there exists
	if binPath, err := exec.LookPath(binFilename); err == nil && binPath != "" {
		if _, err = os.Stat(binPath); err == nil {
			return binPath, nil
		}
	}

	// get the current process path
	if binPath, err := os.Executable(); err == nil {
		return binPath, nil
	}

	return "", errors.New("theo binary not found, please add it to a PATH location")
}
