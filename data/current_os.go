package data

import (
	"github.com/arelate/vangogh_local_data"
	"runtime"
)

var goOperatingSystems = map[string]vangogh_local_data.OperatingSystem{
	"windows": vangogh_local_data.Windows,
	"darwin":  vangogh_local_data.MacOS,
	"linux":   vangogh_local_data.Linux,
}

func CurrentOS() vangogh_local_data.OperatingSystem {
	if os, ok := goOperatingSystems[runtime.GOOS]; ok {
		return os
	} else {
		panic("unsupported operating system")
	}
}
