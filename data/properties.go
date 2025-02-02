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
)

func AllProperties() []string {
	return []string{
		ServerConnectionProperties,
		BundleNameProperty,
		TitleProperty,
		SlugProperty,
	}
}
