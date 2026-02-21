package data

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

const theoDirname = "theo"

const (
	inventoryExt = ".json"
)

const (
	Backups       pathways.AbsDir = "backups"
	Downloads     pathways.AbsDir = "downloads"
	InstalledApps pathways.AbsDir = "installed-apps"
	SteamApps     pathways.AbsDir = "steam-apps" // InstalledApps
	Logs          pathways.AbsDir = "logs"
	Metadata      pathways.AbsDir = "metadata"
	Wine          pathways.AbsDir = "wine"
	SteamCmd      pathways.AbsDir = "steamcmd"
	Temp          pathways.AbsDir = "_temp"
)

const (
	Redux              pathways.RelDir = "_redux"               // Metadata
	ProductDetails     pathways.RelDir = "product-details"      // Metadata
	ManualUrlChecksums pathways.RelDir = "manual-url-checksums" // Metadata
	SteamAppInfo       pathways.RelDir = "steam-appinfo"        // Metadata
	Inventory          pathways.RelDir = "_inventory"           // InstalledApps
	PrefixArchive      pathways.RelDir = "_prefix-archive"      // Backups
	BinDownloads       pathways.RelDir = "_downloads"           // Wine, SteamCmd
	BinUnpacks         pathways.RelDir = "_binaries"            // Wine, SteamCmd
	Prefixes           pathways.RelDir = "_prefixes"            // Wine
	SteamPrefixes      pathways.RelDir = "_steam-prefixes"      // Wine
	UmuConfigs         pathways.RelDir = "_umu-configs"         // Wine
)

var steamCmdBinary = map[vangogh_integration.OperatingSystem]string{
	vangogh_integration.MacOS: "steamcmd.sh",
	vangogh_integration.Linux: "steamcmd.sh",
}

var Pwd pathways.Pathway

func InitPathways() error {
	udhd, err := UserDataHomeDir()
	if err != nil {
		return err
	}

	rootDir := filepath.Join(udhd, theoDirname)
	if _, err = os.Stat(rootDir); os.IsNotExist(err) {
		if err = os.MkdirAll(rootDir, 0755); err != nil {
			return err
		}
	}

	Pwd, err = pathways.NewRoot(rootDir)
	if err != nil {
		return err
	}

	for _, ad := range []pathways.AbsDir{Backups, Metadata, Downloads, InstalledApps, SteamApps, Wine, SteamCmd, Logs, Temp} {
		absDir := filepath.Join(rootDir, string(ad))
		if _, err = os.Stat(absDir); os.IsNotExist(err) {
			if err = os.MkdirAll(absDir, 0755); err != nil {
				return err
			}
		}
	}

	for rd, ads := range map[pathways.RelDir][]pathways.AbsDir{
		PrefixArchive:      {Backups},
		Redux:              {Metadata},
		ProductDetails:     {Metadata},
		ManualUrlChecksums: {Metadata},
		Inventory:          {InstalledApps},
		BinUnpacks:         {Wine, SteamCmd},
		BinDownloads:       {Wine, SteamCmd},
		Prefixes:           {Wine},
		SteamPrefixes:      {Wine},
		UmuConfigs:         {Wine},
	} {
		for _, ad := range ads {
			absRelDir := filepath.Join(rootDir, string(ad), string(rd))
			if _, err = os.Stat(absRelDir); os.IsNotExist(err) {
				if err = os.MkdirAll(absRelDir, 0755); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func GetPrefixName(id string, rdx redux.Readable) (string, error) {
	if slug, ok := rdx.GetLastVal(vangogh_integration.SlugProperty, id); ok && slug != "" {
		return slug, nil
	} else {
		return "", errors.New("product slug is undefined: " + id)
	}
}

func OsLangCode(operatingSystem vangogh_integration.OperatingSystem, langCode string) string {
	return strings.Join([]string{operatingSystem.String(), langCode}, "-")
}

func AbsPrefixDir(id string, rdx redux.Readable) (string, error) {
	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", err
	}

	prefixesDir := Pwd.AbsRelDirPath(Prefixes, Wine)

	prefixName, err := GetPrefixName(id, rdx)
	if err != nil {
		return "", err
	}

	return filepath.Join(prefixesDir, prefixName), nil
}

func AbsSteamPrefixDir(steamAppId string) (string, error) {

	appInfoDir := Pwd.AbsRelDirPath(SteamAppInfo, Metadata)

	kvAppInfo, err := kevlar.New(appInfoDir, steam_vdf.Ext)
	if err != nil {
		return "", err
	}

	appInfoRc, err := kvAppInfo.Get(steamAppId)
	if err != nil {
		return "", err
	}

	defer appInfoRc.Close()

	appInfoKv, err := steam_vdf.ReadText(appInfoRc)
	if err != nil {
		return "", err
	}

	steamPrefixesDir := Pwd.AbsRelDirPath(SteamPrefixes, Wine)

	var appInfoName string
	if ain, ok := appInfoKv.Val(steamAppId, "common", "name"); ok {
		appInfoName = ain
	}

	if appInfoName == "" {
		return "", errors.New("empty appinfo name")
	}

	return filepath.Join(steamPrefixesDir, pathways.Sanitize(appInfoName)), nil
}

func AbsInventoryFilename(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {
	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", err
	}

	osLangInventoryDir := filepath.Join(Pwd.AbsRelDirPath(Inventory, InstalledApps), OsLangCode(operatingSystem, langCode))

	if slug, ok := rdx.GetLastVal(vangogh_integration.SlugProperty, id); ok && slug != "" {
		return filepath.Join(osLangInventoryDir, slug+inventoryExt), nil
	} else {
		return "", errors.New("product slug is undefined: " + id)
	}
}

func RelToUserDataHome(path string) (string, error) {
	udhd, err := UserDataHomeDir()
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(path, udhd) {
		return strings.Replace(path, udhd, "~Data", 1), nil
	} else {
		return path, nil
	}
}

func AbsSteamCmdBinPath(operatingSystem vangogh_integration.OperatingSystem) (string, error) {
	switch operatingSystem {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		steamCmdBinariesDir := Pwd.AbsRelDirPath(BinUnpacks, SteamCmd)
		osSteamCmdBinariesDir := filepath.Join(steamCmdBinariesDir, operatingSystem.String())
		return filepath.Join(osSteamCmdBinariesDir, steamCmdBinary[operatingSystem]), nil
	default:
		return "", operatingSystem.ErrUnsupported()
	}
}

func AbsSteamAppInstallDir(steamAppId string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {

	if err := rdx.MustHave(vangogh_integration.TitleProperty); err != nil {
		return "", err
	}

	var steamAppName string
	if san, ok := rdx.GetLastVal(vangogh_integration.TitleProperty, steamAppId); ok && san != "" {
		steamAppName = san
	}

	if steamAppName == "" {
		return "", errors.New("cannot resolve Steam app name for " + steamAppId)
	}

	steamAppsDir := Pwd.AbsDirPath(SteamApps)

	return filepath.Join(steamAppsDir, operatingSystem.String(), pathways.Sanitize(steamAppName)), nil
}
