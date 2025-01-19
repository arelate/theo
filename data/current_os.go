package data

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"runtime"
)

var goOperatingSystems = map[string]vangogh_integration.OperatingSystem{
	"windows": vangogh_integration.Windows,
	"darwin":  vangogh_integration.MacOS,
	"linux":   vangogh_integration.Linux,
}

func CurrentOs() vangogh_integration.OperatingSystem {
	if os, ok := goOperatingSystems[runtime.GOOS]; ok {
		return os
	} else {
		panic("unsupported operating system")
	}
}
