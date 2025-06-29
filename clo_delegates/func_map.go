package clo_delegates

import "github.com/arelate/theo/cli"

var FuncMap = map[string]func() []string{
	"prefix-mods":   cli.PrefixMods,
	"wine-programs": cli.WinePrograms,
}
