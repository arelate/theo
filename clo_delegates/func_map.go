package clo_delegates

import (
	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/cli"
	"github.com/arelate/theo/data"
)

var FuncMap = map[string]func() []string{
	"prefix-mods":           cli.PrefixMods,
	"wine-programs":         wine_integration.WinePrograms,
	"wine-binaries-codes":   wine_integration.WineBinariesCodes,
	"operating-systems":     vangogh_integration.OperatingSystemsCloValues,
	"language-codes":        gog_integration.LanguageCodesCloValues,
	"download-types":        vangogh_integration.DownloadTypesCloValues,
	"steam-proton-runtimes": wine_integration.AllSteamProtonRuntimes,
	"origins":               data.AllOrigins,
}
