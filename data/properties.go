package data

import "github.com/arelate/southern_light/vangogh_integration"

const (
	ServerConnectionProperties = "server-connection"

	ServerProtocolProperty = "server-protocol"
	ServerAddressProperty  = "server-address"
	ServerPortProperty     = "server-port"
	ServerUsernameProperty = "server-username"
	ServerPasswordProperty = "server-password"

	PrefixEnvProperty = "prefix-env"
	PrefixExeProperty = "prefix-exe"
	PrefixArgProperty = "prefix-arg"

	GitHubReleasesUpdatedProperty = "github-releases-updated"

	InstallInfoProperty = "install-info"

	InstallDateProperty = "install-date"
	LastRunDateProperty = "last-run-date"
)

func AllProperties() []string {
	return []string{
		ServerConnectionProperties,
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
		InstallInfoProperty,
		PrefixEnvProperty,
		PrefixExeProperty,
		InstallDateProperty,
		LastRunDateProperty,
	}
}
