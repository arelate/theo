package cli

import (
	"github.com/boggydigital/nod"
	"net/url"
	"runtime/debug"
)

var (
	GitTag string
)

func VersionHandler(_ *url.URL) error {
	va := nod.Begin("checking theo version...")
	defer va.Done()

	if GitTag == "" {
		summary := make(map[string][]string)
		if bi, ok := debug.ReadBuildInfo(); ok {
			values := []string{bi.Main.Version, bi.Main.Path, bi.GoVersion}
			for _, value := range values {
				if value != "" {
					summary["version info:"] = append(summary["version info:"], value)
				}
			}
			va.EndWithSummary("", summary)
		} else {
			va.EndWithResult("unknown version")
		}
	} else {
		va.EndWithResult(GitTag)
	}
	return nil
}
