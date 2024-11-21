package data

const (
	SetupProperties = "setup"

	VangoghProtocolProperty  = "vangogh-protocol"
	VangoghAddressProperty   = "vangogh-address"
	VangoghPortProperty      = "vangogh-port"
	VangoghUsernameProperty  = "vangogh-username"
	VangoghPasswordProperty  = "vangogh-password"
	InstallationPathProperty = "installation-path"

	BundleNameProperty = "bundle-name"
	TitleProperty      = "title"
)

func AllProperties() []string {
	return []string{
		SetupProperties,
		BundleNameProperty,
		TitleProperty,
	}
}
