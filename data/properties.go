package data

import "github.com/arelate/southern_light/vangogh_integration"

const (
	ServerConnectionProperties = "server-connection"

	InstallInfoProperty          = "install-info"
	InstallDateProperty          = "install-date"
	LastRunDateProperty          = "last-run-date"
	PlaytimeMinutesProperty      = "playtime-minutes"
	TotalPlaytimeMinutesProperty = "total-playtime-minutes"

	PrefixEnvProperty = "prefix-env"
	PrefixExeProperty = "prefix-exe"
	PrefixArgProperty = "prefix-arg"

	WineBinariesVersionsProperty = "wine-binaries-versions"
)

const (
	ServerProtocolProperty = "server-protocol"
	ServerAddressProperty  = "server-address"
	ServerPortProperty     = "server-port"
	ServerUsernameProperty = "server-username"
	ServerSessionToken     = "server-session-token"
	ServerSessionExpires   = "server-session-expires"
)

func AllProperties() []string {
	return []string{
		vangogh_integration.TitleProperty,
		vangogh_integration.SlugProperty,
		vangogh_integration.SteamAppIdProperty,
		vangogh_integration.OperatingSystemsProperty,
		vangogh_integration.DevelopersProperty,
		vangogh_integration.PublishersProperty,
		vangogh_integration.VerticalImageProperty,
		vangogh_integration.ImageProperty,
		vangogh_integration.HeroProperty,
		vangogh_integration.LogoProperty,
		vangogh_integration.IconProperty,
		vangogh_integration.IconSquareProperty,
		vangogh_integration.BackgroundProperty,
		ServerConnectionProperties,
		InstallInfoProperty,
		InstallDateProperty,
		LastRunDateProperty,
		PlaytimeMinutesProperty,
		TotalPlaytimeMinutesProperty,
		PrefixEnvProperty,
		PrefixExeProperty,
		PrefixArgProperty,
		WineBinariesVersionsProperty,
	}
}
