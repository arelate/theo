package data

import (
	"runtime"

	"github.com/arelate/southern_light/vangogh_integration"
)

func CurrentOs() vangogh_integration.OperatingSystem {
	switch runtime.GOOS {
	case "windows":
		return vangogh_integration.Windows
	case "darwin":
		return vangogh_integration.MacOS
	case "linux":
		return vangogh_integration.Linux
	default:
		panic("current os is not supported")
	}
}
