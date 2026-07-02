package data

import "github.com/arelate/southern_light/vangogh_integration"

const (
	VangoghUrlProperty            = "vangogh-url"
	VangoghUsernameProperty       = "vangogh-username"
	VangoghSessionTokenProperty   = "vangogh-session-token"
	VangoghSessionExpiresProperty = "vangogh-session-expires"

	SteamUsernameProperty = "steam-username"

	InstallInfoProperty          = "install-info"
	InstallDateProperty          = "install-date"
	LastRunDateProperty          = "last-run-date"
	PlaytimeMinutesProperty      = "playtime-minutes"
	TotalPlaytimeMinutesProperty = "total-playtime-minutes"

	LaunchOptionsExeProperty = "launch-options-exe"
	LaunchOptionsArgProperty = "launch-options-arg"
	LaunchOptionsEnvProperty = "launch-options-env"

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
			vangogh_integration.GogTitleProperty,
			vangogh_integration.SteamTitleProperty,
			vangogh_integration.EgsTitleProperty,
			vangogh_integration.EgsMainGameProperty,
			vangogh_integration.GogSteamAppIdProperty,
			vangogh_integration.GogOperatingSystemsProperty,
			vangogh_integration.GogDevelopersProperty,
			vangogh_integration.GogPublishersProperty,
			vangogh_integration.GogVerticalImageProperty,
			vangogh_integration.GogImageProperty,
			vangogh_integration.GogHeroProperty,
			vangogh_integration.GogLogoProperty,
			vangogh_integration.GogIconProperty,
			vangogh_integration.GogIconSquareProperty,
			vangogh_integration.GogBackgroundProperty,
			vangogh_integration.GogRequiresGamesProperty,
			vangogh_integration.GogBundleNameProperty,
			InstallInfoProperty,
			InstallDateProperty,
			LastRunDateProperty,
			PlaytimeMinutesProperty,
			TotalPlaytimeMinutesProperty,
			LaunchOptionsExeProperty,
			LaunchOptionsArgProperty,
			LaunchOptionsEnvProperty,
			WineBinariesVersionsProperty,
		}...)

	return ap
}
