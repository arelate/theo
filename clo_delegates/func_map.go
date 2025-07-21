package clo_delegates

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/cli"
)

var FuncMap = map[string]func() []string{
	"prefix-mods":       cli.PrefixMods,
	"wine-programs":     cli.WinePrograms,
	"operating-systems": vangogh_integration.OperatingSystemsCloValues,
	"language-codes":    vangogh_integration.DownloadsLayoutsCloValues,
	"download-types":    vangogh_integration.DownloadsLayoutsCloValues,
}
