package cli

import (
	"net/url"
	"os"
	"strings"

	"github.com/arelate/southern_light/steamcmd"
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

func Uninstall(id string, request *InstallInfo, purge bool) error {

	ua := nod.Begin("uninstalling %s...", id)
	defer ua.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	if !request.force {
		ua.EndWithResult("uninstall requires -force parameter")
		return nil
	}

	installInfo, err := matchInstalledInfo(id, request, rdx)
	if err != nil {
		return err
	}

	var installedAppDir string
	installedAppDir, err = originOsInstalledPath(id, installInfo, rdx)
	if err != nil {
		return err
	}

	switch installInfo.Origin {
	case data.VangoghGogOrigin:
		if err = osUninstallProduct(id, installInfo, rdx); err != nil {
			return err
		}

		var absInventoryFilename string
		absInventoryFilename, err = data.AbsInventoryFilename(id, installInfo.LangCode, installInfo.OperatingSystem, rdx)
		if err != nil {
			return err
		}

		if _, err = os.Stat(absInventoryFilename); err == nil {
			if err = os.Remove(absInventoryFilename); err != nil {
				return err
			}
		}
	case data.SteamOrigin:

		var absSteamCmdPath string
		absSteamCmdPath, err = data.AbsSteamCmdBinPath(data.CurrentOs())
		if err != nil {
			return err
		}

		if err = steamcmd.AppUninstall(absSteamCmdPath, id, installInfo.OperatingSystem, installedAppDir); err != nil {
			return err
		}
	default:
		return installInfo.Origin.ErrUnsupportedOrigin()
	}

	if purge {
		if _, err = os.Stat(installedAppDir); err == nil {
			if err = os.RemoveAll(installedAppDir); err != nil {
				return err
			}
		}

		// account for macOS bundle name
		if installInfo.OperatingSystem == vangogh_integration.MacOS {
			if bundleName, ok := rdx.GetLastVal(data.BundleNameProperty, id); ok && bundleName != "" && !strings.Contains(bundleName, "/") {
				installedAppParentDir := strings.TrimSuffix(installedAppDir, bundleName)
				if _, err = os.Stat(installedAppParentDir); err == nil {
					if err = os.RemoveAll(installedAppParentDir); err != nil {
						return err
					}
				}
			}
		}
	}

	if err = unpinInstallInfo(id, installInfo, rdx); err != nil {
		return err
	}

	if err = removeSteamShortcut(id, rdx); err != nil {
		return err
	}

	return nil
}

func osUninstallProduct(id string, ii *InstallInfo, rdx redux.Writeable) error {

	oupa := nod.Begin(" uninstalling %s %s-%s...", id, ii.OperatingSystem, ii.LangCode)
	defer oupa.Done()

	if err := removeInventoriedFiles(id, ii, rdx); err != nil {
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
