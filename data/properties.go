package data

import "github.com/arelate/southern_light/vangogh_integration"

const (
	VangoghUrlProperty            = "vangogh-url"
	VangoghUsernameProperty       = "vangogh-username"
	VangoghSessionTokenProperty   = "vangogh-session-token"
	VangoghSessionExpiresProperty = "vangogh-session-expires"

	SteamUsernameProperty = "steam-username"

	BundleNameProperty = "bundle-name"

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

func VangoghProperties() []string {
	return []string{
		VangoghUrlProperty,
		VangoghUsernameProperty,
		VangoghSessionTokenProperty,
		VangoghSessionExpiresProperty,
	}
}

func SteamProperties() []string {
	return []string{
		SteamUsernameProperty,
	}
}

func AllProperties() []string {
	ap := VangoghProperties()
	ap = append(ap, SteamProperties()...)
	ap = append(ap,
		[]string{
			vangogh_integration.TitleProperty,
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
			BundleNameProperty,
			InstallInfoProperty,
			InstallDateProperty,
			LastRunDateProperty,
			PlaytimeMinutesProperty,
			TotalPlaytimeMinutesProperty,
			PrefixEnvProperty,
			PrefixExeProperty,
			PrefixArgProperty,
			WineBinariesVersionsProperty,
		}...)

	return ap
}
