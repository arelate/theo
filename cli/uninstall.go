package cli

import (
	"net/url"
	"os"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func UninstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	var langCode string
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

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	if !ii.force {
		ua.EndWithResult("uninstall requires -force parameter")
		return nil
	}

	if err = resolveInstallInfo(id, ii, nil, rdx, installedOperatingSystem, installedLangCode); err != nil {
		return err
	}

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		var installInfo *InstallInfo
		installInfo, _, err = matchInstallInfo(ii, installedInfoLines...)
		if err != nil {
			return err
		}

		if installInfo == nil {
			ua.EndWithResult("no install info found for %s %s-%s", id, ii.OperatingSystem, ii.LangCode)
			return nil
		}

	}

	if err = osUninstallProduct(id, ii, rdx); err != nil {
		return err
	}

	absInventoryFilename, err := data.AbsInventoryFilename(id, ii.LangCode, ii.OperatingSystem, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absInventoryFilename); err == nil {
		if err = os.Remove(absInventoryFilename); err != nil {
			return err
		}
	}

	if err = unpinInstallInfo(id, ii, rdx); err != nil {
		return err
	}

	if err = removeSteamShortcut(rdx, id); err != nil {
		return err
	}

	return nil

}

func osUninstallProduct(id string, ii *InstallInfo, rdx redux.Writeable) error {

	oupa := nod.Begin(" uninstalling %s %s-%s...", id, ii.OperatingSystem, ii.LangCode)
	defer oupa.Done()

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		if err := removeInventoriesFiles(id, ii.LangCode, ii.OperatingSystem, rdx); err != nil {
			return err
		}
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:

			if err := removeProductPrefix(id, ii.LangCode, rdx, ii.force); err != nil {
				return err
			}

			if err := prefixDeleteProperty(id, ii.LangCode, data.PrefixEnvProperty, rdx, ii.force); err != nil {
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
