package cli

import (
	"net/url"

	"github.com/arelate/southern_light/vangogh_integration"
)

func PresetLaunchOptionsHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	ii := new(InstallInfo{
		OperatingSystem: vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty)),
		LangCode:        q.Get(vangogh_integration.LanguageCodeProperty),
	})

	reset := q.Has("reset")

	return PresetLaunchOptions(id, ii, reset)
}

func PresetLaunchOptions(id string, request *InstallInfo, reset bool) error {
	return nil
}
