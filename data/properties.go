package data

const (
	ServerConnectionProperties = "server-connection"

	ServerProtocolProperty = "server-protocol"
	ServerAddressProperty  = "server-address"
	ServerPortProperty     = "server-port"
	ServerUsernameProperty = "server-username"
	ServerPasswordProperty = "server-password"

	BundleNameProperty = "bundle-name"
	TitleProperty      = "title"
	SlugProperty       = "slug"

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
		TitleProperty,
		SlugProperty,
		InstallParametersProperty,
		PrefixEnvProperty,
		PrefixExePathProperty,
		InstallDateProperty,
		LastRunDateProperty,
	}
}
