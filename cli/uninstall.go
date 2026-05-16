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

	switch purge {
	case true:
		if err = originPurgeInstallation(id, installInfo, rdx); err != nil {
			return err
		}
	default:
		if err = originUninstall(id, installInfo, rdx); err != nil {
			return err
		}
	}

	if err = originPostUninstall(id, installInfo, rdx, purge); err != nil {
		return err
	}

	if err = LaunchOptions(id, installInfo, new(execTask), true); err != nil {
		return err
	}

	if err = unpinInstallInfo(id, installInfo, rdx); err != nil {
		return err
	}

	if err = removeSteamShortcut(id, rdx); err != nil {
		return err
	}

	return nil
}

func originUninstall(id string, installInfo *InstallInfo, rdx redux.Writeable) error {

	installedAppDir, err := originOsInstalledPath(id, installInfo, rdx)
	if err != nil {
		return err
	}

	switch installInfo.Origin {
	case data.VangoghOrigin:
		if err = vangoghUninstallProduct(id, installInfo, rdx); err != nil {
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
	case data.EpicGamesOrigin:

		var originData *data.OriginData
		originData, err = originGetData(id, installInfo, rdx, false)
		if err != nil {
			return err
		}

		if err = egsUninstall(id, installInfo, originData, rdx); err != nil {
			return err
		}

	default:
		return installInfo.Origin.ErrUnsupportedOrigin()
	}

	return nil
}

func originPurgeInstallation(id string, installInfo *InstallInfo, rdx redux.Readable) error {

	installedAppDir, err := originOsInstalledPath(id, installInfo, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(installedAppDir); err == nil {
		if err = os.RemoveAll(installedAppDir); err != nil {
			return err
		}
	}

	// account for macOS bundle title
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

	return nil
}

func originPostUninstall(id string, ii *InstallInfo, rdx redux.Writeable, purge bool) error {
	switch ii.Origin {
	case data.EpicGamesOrigin:
		switch purge {
		case true:
			return nil
		default:
			// purge already removed main product installation directory where DLC was installed,
			// so only uninstall DLC if purge was not set
			return egsUninstallDownloadableContent(id, ii, rdx)
		}
	default:
		return nil
	}
}
