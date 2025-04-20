package data

import "github.com/arelate/southern_light/vangogh_integration"

const (
	ServerConnectionProperties = "server-connection"

	ServerProtocolProperty = "server-protocol"
	ServerAddressProperty  = "server-address"
	ServerPortProperty     = "server-port"
	ServerUsernameProperty = "server-username"
	ServerPasswordProperty = "server-password"

	BundleNameProperty = "bundle-name"

	PrefixEnvProperty     = "prefix-env"
	PrefixExePathProperty = "prefix-exe-path"

	GitHubReleasesUpdatedProperty = "github-releases-updated"

	InstallParametersProperty = "install-parameters"
	KeepDownloadsProperty     = "keep-downloads"
	NoSteamShortcutProperty   = "no-steam-shortcut"

	InstallDateProperty = "install-date"
	LastRunDateProperty = "last-run-date"
)

func AllProperties() []string {
	return []string{
		ServerConnectionProperties,
		BundleNameProperty,
		vangogh_integration.TitleProperty,
		vangogh_integration.SlugProperty,
		vangogh_integration.SteamAppIdProperty,
		InstallParametersProperty,
		PrefixEnvProperty,
		PrefixExePathProperty,
		InstallDateProperty,
		LastRunDateProperty,
	}
}
