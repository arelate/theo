package data

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"os"
	"os/exec"
)

func TheoExecutable() (string, error) {

	binFilename := "theo"

	switch CurrentOS() {
	case vangogh_integration.Windows:
		binFilename += ".exe"
	case vangogh_integration.Linux:
		fallthrough
	case vangogh_integration.MacOS:
	// do nothing
	default:
		return "", errors.New("unsupported operating system")
	}

	if binPath, err := exec.LookPath(binFilename); err == nil && binPath != "" {
		return binPath, nil
	} else if executable, err := os.Executable(); err == nil {
		return executable, nil
	}

	return "", errors.New("theo binary not found, please add it to a PATH location")
}
