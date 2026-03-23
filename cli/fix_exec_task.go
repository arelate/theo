package cli

import (
	"path/filepath"

	"github.com/arelate/southern_light/vangogh_integration"
)

func fixExecTask(id string, operatingSystem vangogh_integration.OperatingSystem, et *execTask) *execTask {
	switch id {
	case "1456460669": // Baldur's Gate 3
		switch operatingSystem {
		case vangogh_integration.MacOS:
			dir, fn := filepath.Split(et.exe)
			if fn == "Baldur's Gate 3" {
				et.exe = filepath.Join(dir, "Baldur's Gate 3 GOG")
			}
		default:
			// do nothing
		}
	default:
		// do nothing
	}

	return et
}
