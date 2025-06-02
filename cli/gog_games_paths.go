package cli

import "github.com/arelate/southern_light/gog_integration"

const (
	gogGamesDir             = "GOG Games"
	gogGameInstallDir       = gogGamesDir + "/*"
	gogGameLnkGlob          = gogGamesDir + "/*/*.lnk"
	gogGameInfoGlobTemplate = gogGamesDir + "/*/" + gog_integration.GogGameInfoFilenameTemplate
)
