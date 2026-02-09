package cli

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/steam_appinfo"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func SteamInstallHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        defaultLangCode,
		DownloadTypes:   []vangogh_integration.DownloadType{vangogh_integration.Installer},
		UseSteamAssets:  true,
		NoSteamShortcut: q.Has("no-steam-shortcut"),
		reveal:          q.Has("reveal"),
		verbose:         q.Has("verbose"),
		force:           q.Has("force"),
	}

	return SteamInstall(id, ii)
}

func SteamInstall(steamAppId string, ii *InstallInfo) error {

	sia := nod.Begin("installing Steam %s for %s...", steamAppId, ii.OperatingSystem)
	defer sia.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.SteamProperties()...)
	if err != nil {
		return err
	}

	steamAppInfoDir := data.Pwd.AbsRelDirPath(data.SteamAppInfo, data.Metadata)
	kvSteamAppInfo, err := kevlar.New(steamAppInfoDir, steam_vdf.Ext)
	if err != nil {
		return err
	}

	var username string
	if un, ok := rdx.GetLastVal(data.SteamUsernameProperty, data.SteamUsernameProperty); ok && un != "" {
		username = un
	}

	if err = getSteamAppInfo(steamAppId, username, kvSteamAppInfo, ii.force); err != nil {
		return err
	}

	appInfoRc, err := kvSteamAppInfo.Get(steamAppId)
	if err != nil {
		return err
	}
	defer appInfoRc.Close()

	appInfoKeyValues, err := steam_vdf.ReadText(appInfoRc)
	if err != nil {
		return err
	}

	appInfo, err := steam_appinfo.AppInfoVdf(appInfoKeyValues)
	if err != nil {
		return err
	}

	productDetails := steamAppInfoProductDetails(appInfo)

	if err = resolveInstallInfo(steamAppId, ii, productDetails, rdx, currentOsThenWindows); err != nil {
		return err
	}

	if ii.OperatingSystem == vangogh_integration.Windows && data.CurrentOs() != vangogh_integration.Windows {
		if err = steamPrefixInit(steamAppId, ii.verbose); err != nil {
			return err
		}
	}

	if err = steamUpdateApp(steamAppId, appInfo.Common.Name, username, ii.OperatingSystem, appInfo.Config.InstallDir); err != nil {
		return err
	}

	steamAppsDir := data.Pwd.AbsDirPath(data.SteamApps)
	absInstallDir := filepath.Join(steamAppsDir, ii.OperatingSystem.String(), appInfo.Config.InstallDir)

	sgo := &steamGridOptions{
		useSteamAssets: true,
		steamRun:       true,
		name:           appInfo.Common.Name,
		installDir:     absInstallDir,
		logoPosition:   nil,
	}

	if err = SteamShortcut([]string{steamAppId}, nil, false, ii, sgo); err != nil {
		return err
	}

	return nil
}

func getSteamAppInfo(id string, username string, kvSteamAppInfo kevlar.KeyValues, force bool) error {

	scaia := nod.Begin(" getting Steam appinfo for %s...", id)
	defer scaia.Done()

	if kvSteamAppInfo.Has(id) && !force {
		scaia.EndWithResult("already exist")
		return nil
	}

	printedAppInfo, err := steamCmdAppInfoPrint(id, username)
	if err != nil {
		return err
	}

	if err = kvSteamAppInfo.Set(id, strings.NewReader(printedAppInfo)); err != nil {
		return err
	}

	return nil
}

func steamUpdateApp(id, name string, username string, operatingSystem vangogh_integration.OperatingSystem, installDir string) error {

	scaua := nod.Begin("updating and verifying %s (%s) for %s with SteamCMD, please wait...", name, id, operatingSystem)
	defer scaua.Done()

	steamAppsDir := data.Pwd.AbsDirPath(data.SteamApps)
	absInstallDir := filepath.Join(steamAppsDir, operatingSystem.String(), installDir)

	if _, err := os.Stat(absInstallDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absInstallDir, 0755); err != nil {
			return err
		}
	}

	return steamCmdAppUpdate(id, operatingSystem, absInstallDir, username)
}

func steamAppInfoProductDetails(appInfo *steam_appinfo.AppInfo) *vangogh_integration.ProductDetails {

	var operatingSystems []vangogh_integration.OperatingSystem
	if appInfo.Common.OsList != "" {
		operatingSystems = vangogh_integration.ParseManyOperatingSystems(strings.Split(appInfo.Common.OsList, ","))
	} else {
		operatingSystems = append(operatingSystems, vangogh_integration.Windows)
	}

	productDetails := &vangogh_integration.ProductDetails{
		SteamAppId:       appInfo.AppId,
		Title:            appInfo.Common.Name,
		ProductType:      vangogh_integration.GameProductType,
		OperatingSystems: operatingSystems,
		Developers:       []string{appInfo.Extended.Developer},
		Publishers:       []string{appInfo.Extended.Publisher},
	}

	return productDetails
}
