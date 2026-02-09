package cli

import (
	"net/url"

	"github.com/arelate/southern_light/vangogh_integration"
)

func SteamUninstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        defaultLangCode,
		SteamInstall:    true,
		verbose:         q.Has("verbose"),
		force:           q.Has("force"),
	}

	purge := q.Has("purge")

	return SteamUninstall(id, ii, purge)
}

func SteamUninstall(id string, ii *InstallInfo, purge bool) error {
	return nil
}
