package cli

import (
	"fmt"
	"net/url"
	"runtime/debug"
)

var (
	GitTag string
)

func VersionHandler(_ *url.URL) error {
	if GitTag == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			fmt.Println(bi)
		} else {
			fmt.Println("unknown version")
		}
	} else {
		fmt.Println(GitTag)
	}
	return nil
}
