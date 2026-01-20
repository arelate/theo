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

	purge := q.Has("purge")

	return Uninstall(id, ii, purge)
}

func Uninstall(id string, ii *InstallInfo, purge bool) error {

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

	if purge {
		var installedAppDir string
		installedAppDir, err = osInstalledPath(id, ii.LangCode, ii.OperatingSystem, rdx)
		if err != nil {
			return err
		}

		if _, err = os.Stat(installedAppDir); err == nil {
			if err = os.RemoveAll(installedAppDir); err != nil {
				return err
			}
		}
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

	if err := removeInventoriedFiles(id, ii.LangCode, ii.OperatingSystem, rdx); err != nil {
		return err
	}

	switch ii.OperatingSystem {
	case vangogh_integration.Windows:
		switch data.CurrentOs() {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			if err := prefixDeleteProperty(id, ii.LangCode, data.PrefixEnvProperty, rdx, ii.force); err != nil {
				return err
			}
		default:
			return data.CurrentOs().ErrUnsupported()
		}
	default:
		// do nothing
	}

	return nil
}
