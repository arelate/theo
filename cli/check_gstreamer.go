package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/nod"
	"net/url"
	"os"
)

const (
	gstreamerFrameworkPath = "/Library/Frameworks/GStreamer.framework"
)

func CheckGstreamerHandler(u *url.URL) error {
	return CheckGstreamer()
}

func CheckGstreamer() error {
	cga := nod.Begin("checking whether GStreamer.framework is installed...")
	defer cga.EndWithResult("done")

	if data.CurrentOS() != vangogh_local_data.MacOS {
		cga.EndWithResult("skipping. GStreamer is only required on %s", vangogh_local_data.MacOS)
		return nil
	}

	if _, err := os.Stat(gstreamerFrameworkPath); err == nil {
		cga.EndWithResult("found")
		return nil
	} else if os.IsNotExist(err) {
		cga.EndWithResult("not found. Download it at https://gstreamer.freedesktop.org/download")
		return nil
	} else {
		return cga.EndWithError(err)
	}
}
