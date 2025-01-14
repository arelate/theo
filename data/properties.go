package data

const (
	SetupProperties = "setup"

	VangoghProtocolProperty = "vangogh-protocol"
	VangoghAddressProperty  = "vangogh-address"
	VangoghPortProperty     = "vangogh-port"
	VangoghUsernameProperty = "vangogh-username"
	VangoghPasswordProperty = "vangogh-password"

	BundleNameProperty = "bundle-name"
	TitleProperty      = "title"
	SlugProperty       = "slug"

	GitHubReleasesUpdatedProperty = "github-releases-updated"
)

func AllProperties() []string {
	return []string{
		SetupProperties,
		BundleNameProperty,
		TitleProperty,
		SlugProperty,
	}
}
