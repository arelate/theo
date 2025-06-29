package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

func UninstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	langCode := "" // installed info language will be used instead of default
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
		verbose:         q.Has("verbose"),
		force:           q.Has("force"),
	}

	return Uninstall(id, ii)
}

func Uninstall(id string, ii *InstallInfo) error {

	ua := nod.Begin("uninstalling %s...", id)
	defer ua.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	if !ii.force {
		ua.EndWithResult("uninstallation requires force parameter")
		return nil
	}

	if ii.OperatingSystem == vangogh_integration.AnyOperatingSystem {
		iios, err := installedInfoOperatingSystem(id, rdx)
		if err != nil {
			return err
		}

		ii.OperatingSystem = iios
	}

	if ii.LangCode == "" {
		lc, err := installedInfoLangCode(id, ii.OperatingSystem, rdx)
		if err != nil {
			return err
		}

		ii.LangCode = lc
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		installInfo, err := matchInstallInfo(ii, installedInfoLines...)
		if err != nil {
			return err
		}

		if installInfo == nil {
			ua.EndWithResult("on installation info found for %s-%s", id, ii.OperatingSystem, ii.LangCode)
			return nil
		}

	}

	if err = osUninstallProduct(id, ii, rdx); err != nil {
		return err
	}

	if err = unpinInstallInfo(id, ii, rdx); err != nil {
		return err
	}

	if err = removeSteamShortcut(rdx, id); err != nil {
		return err
	}

	return nil

}

func osUninstallProduct(id string, ii *InstallInfo, rdx redux.Readable) error {

	oupa := nod.Begin(" uninstalling %s for %s...", id, ii.OperatingSystem)
	defer oupa.Done()

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		if err := nixUninstallProduct(id, ii.LangCode, ii.OperatingSystem, rdx); err != nil {
			return err
		}
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:

			if err := RemovePrefix(ii.LangCode, ii.force, id); err != nil {
				return err
			}

			if err := DeletePrefixEnv(ii.LangCode, ii.force, id); err != nil {
				return err
			}

		case vangogh_integration.Windows:
			if err := windowsUninstallProduct(id, ii.LangCode, rdx); err != nil {
				return err
			}
		default:
			return currentOs.ErrUnsupported()
		}
	default:
		return ii.OperatingSystem.ErrUnsupported()
	}

	return nil
}
