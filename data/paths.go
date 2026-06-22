package data

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
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
	SteamApps     pathways.AbsDir = "steam-apps"
	EgsApps       pathways.AbsDir = "egs-apps"
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
	Cookies            pathways.RelDir = "_cookies"             // Metadata
	Tokens             pathways.RelDir = "_tokens"              // Metadata
	AvailableProducts  pathways.RelDir = "available-products"   // Metadata
	GameAssets         pathways.RelDir = "game-assets"          // Metadata
	CatalogItems       pathways.RelDir = "catalog-items"        // Metadata
	GameManifests      pathways.RelDir = "game-manifests"       // Metadata
	Manifests          pathways.RelDir = "manifests"            // Metadata
	Inventory          pathways.RelDir = "_inventory"           // InstalledApps
	PrefixArchive      pathways.RelDir = "_prefix-archive"      // Backups
	BinDownloads       pathways.RelDir = "_downloads"           // Wine, SteamCmd
	BinUnpacks         pathways.RelDir = "_binaries"            // Wine, SteamCmd
	Prefixes           pathways.RelDir = "_prefixes"            // Wine
	SteamPrefixes      pathways.RelDir = "_steam-prefixes"      // Wine
	EgsPrefixes        pathways.RelDir = "_egs-prefixes"        // Wine
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
		Cookies:            {Metadata},
		Tokens:             {Metadata},
		AvailableProducts:  {Metadata},
		GameAssets:         {Metadata},
		CatalogItems:       {Metadata},
		GameManifests:      {Metadata},
		Manifests:          {Metadata},
		Inventory:          {InstalledApps},
		BinUnpacks:         {Wine, SteamCmd},
		BinDownloads:       {Wine, SteamCmd},
		Prefixes:           {Wine},
		SteamPrefixes:      {Wine},
		EgsPrefixes:        {Wine},
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

func GetTitleProperty(id string, rdx redux.Readable) (string, error) {
	titleProperties := []string{
		vangogh_integration.GogTitleProperty,
		vangogh_integration.SteamTitleProperty,
		vangogh_integration.EgsTitleProperty,
	}

	if err := rdx.MustHave(titleProperties...); err != nil {
		return "", err
	}

	for _, tp := range titleProperties {
		if title, ok := rdx.GetLastVal(tp, id); ok && title != "" {
			return title, nil
		}
	}

	return "", errors.New("title not found for " + id)
}

func OsLangCode(operatingSystem vangogh_integration.OperatingSystem, langCode string) string {
	return strings.Join([]string{operatingSystem.String(), langCode}, "-")
}

func AppOsLangCode(id string, operatingSystem vangogh_integration.OperatingSystem, langCode string) string {
	return strings.Join([]string{id, operatingSystem.String(), langCode}, "-")
}

func AbsPrefixDir(id string, origin Origin, rdx redux.Readable) (string, error) {

	var prefixesDir string
	switch origin {
	case VangoghOrigin:
		prefixesDir = Pwd.AbsRelDirPath(Prefixes, Wine)
	case SteamOrigin:
		prefixesDir = Pwd.AbsRelDirPath(SteamPrefixes, Wine)
	case EpicGamesOrigin:
		prefixesDir = Pwd.AbsRelDirPath(EgsPrefixes, Wine)
	default:
		return "", origin.ErrUnsupportedOrigin()
	}

	title, err := GetTitleProperty(id, rdx)
	if err != nil {
		return "", err
	}

	return filepath.Join(prefixesDir, pathways.Sanitize(title)), nil
}

func AbsInventoryFilename(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {

	osLangInventoryDir := filepath.Join(Pwd.AbsRelDirPath(Inventory, InstalledApps), OsLangCode(operatingSystem, langCode))

	title, err := GetTitleProperty(id, rdx)
	if err != nil {
		return "", err
	}

	return filepath.Join(osLangInventoryDir, pathways.Sanitize(title)+inventoryExt), nil
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

	if err := rdx.MustHave(vangogh_integration.SteamTitleProperty); err != nil {
		return "", err
	}

	var steamAppName string
	if san, ok := rdx.GetLastVal(vangogh_integration.SteamTitleProperty, steamAppId); ok && san != "" {
		steamAppName = san
	}

	if steamAppName == "" {
		return "", errors.New("Steam app name not found for " + steamAppId)
	}

	steamAppsDir := Pwd.AbsDirPath(SteamApps)

	return filepath.Join(steamAppsDir, operatingSystem.String(), pathways.Sanitize(steamAppName)), nil
}

func AbsChunksDownloadDir(appName string, operatingSystem vangogh_integration.OperatingSystem) string {
	return filepath.Join(Pwd.AbsDirPath(Downloads), fmt.Sprintf("%s-%s", appName, operatingSystem))
}

func AbsReduxDir() string {
	return Pwd.AbsRelDirPath(Redux, Metadata)
}
