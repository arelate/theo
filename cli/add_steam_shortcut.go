package cli

import (
	"errors"
	"fmt"
	"github.com/arelate/southern_light/steam_integration"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	loginUsersFilename = "loginusers.vdf"
	shortcutsFilename  = "shortcuts.vdf"
)

const (
	runLaunchOptionsTemplate     = "run %s"
	wineRunLaunchOptionsTemplate = "wine-run %s"
)

func AddSteamShortcutHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	wine := q.Has("wine")
	force := q.Has("force")

	return AddSteamShortcut(langCode, wine, force, ids...)
}

func AddSteamShortcut(langCode string, wine, force bool, ids ...string) error {
	assa := nod.Begin("adding Steam shortcuts for %s...", strings.Join(ids, ","))
	defer assa.EndWithResult("done")

	ok, err := steamStateDirExist()
	if err != nil {
		return assa.EndWithError(err)
	}

	if !ok {
		assa.EndWithResult("Steam state dir not found")
		return nil
	}

	loginUsers, err := getSteamLoginUsers()
	if err != nil {
		return assa.EndWithError(err)
	}

	for _, loginUser := range loginUsers {
		if err := addSteamShortcutsForUser(loginUser, langCode, wine, force, ids...); err != nil {
			return assa.EndWithError(err)
		}
	}

	return nil
}

func addSteamShortcutsForUser(loginUser string, langCode string, wine, force bool, ids ...string) error {

	asfua := nod.Begin(" adding Steam user %s shortcuts for %s...",
		loginUser,
		strings.Join(ids, ","))
	defer asfua.EndWithResult("done")

	kvUserShortcuts, err := readUserShortcuts(loginUser)
	if err != nil {
		return asfua.EndWithError(err)
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return asfua.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir,
		data.TitleProperty,
		data.BundleNameProperty,
		data.SetupProperties,
		data.SlugProperty)

	if err != nil {
		return asfua.EndWithError(err)
	}

	theoExecutable, err := data.TheoExecutable()
	if err != nil {
		return asfua.EndWithError(err)
	}

	for _, id := range ids {

		var title string
		if tp, ok := rdx.GetLastVal(data.TitleProperty, id); ok && tp != "" {
			title = tp
		} else {
			return asfua.EndWithError(errors.New("product is missing title"))
		}

		startDir, err := getStartDir(id, langCode, rdx, wine)
		if err != nil {
			return asfua.EndWithError(err)
		}

		shortcutId := steam_integration.ShortcutAppId(theoExecutable, title)
		iconPath := getGridIconPath(loginUser, shortcutId)

		launchOptionsTemplate := runLaunchOptionsTemplate
		if wine {
			launchOptionsTemplate = wineRunLaunchOptionsTemplate
		}

		launchOptions := fmt.Sprintf(launchOptionsTemplate, id)
		if langCode != "" {
			launchOptions += fmt.Sprintf(" -lang-code %s", langCode)
		}

		if changed, err := addNonSteamAppShortcut(shortcutId, title, theoExecutable, iconPath, startDir, launchOptions, kvUserShortcuts, force); err != nil {
			return asfua.EndWithError(err)
		} else if changed {
			if err := writeUserShortcuts(loginUser, kvUserShortcuts); err != nil {
				return err
			}
		}

		metadata, err := getTheoMetadata(id, force)
		if err != nil {
			return asfua.EndWithError(err)
		}

		if err := downloadSteamGridImages(loginUser, shortcutId, &metadata.Images, rdx, force); err != nil {
			return asfua.EndWithError(err)
		}
	}

	return nil
}

func getStartDir(id, langCode string, rdx kevlar.ReadableRedux, wine bool) (string, error) {

	startDir := ""

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return "", err
	}

	switch wine {
	case true:

		prefixName, err := data.GetPrefixName(id, langCode, rdx)
		if err != nil {
			return "", err
		}

		absPrefixDir, err := data.GetAbsPrefixDir(prefixName)
		if err != nil {
			return "", err
		}

		startDir = filepath.Join(absPrefixDir, data.RelPfxDriveCDir)

	case false:

		var bundleName string
		if bn, ok := rdx.GetLastVal(data.BundleNameProperty, id); ok && bn != "" {
			bundleName = bn
		} else {
			return "", errors.New("product is missing bundle name")
		}

		osLangCodeDir := data.OsLangCodeDir(data.CurrentOS(), langCode)
		startDir = filepath.Join(installedAppsDir, osLangCodeDir, bundleName)
	}

	return startDir, nil
}

func downloadSteamGridImages(loginUser string, shortcutId uint32, imagesMetadata *vangogh_integration.TheoImages, rdx kevlar.ReadableRedux, force bool) error {

	dsgia := nod.Begin(" downloading Steam Grid images...")
	defer dsgia.EndWithResult("done")

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return dsgia.EndWithError(err)
	}

	absSteamGridPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid")
	dc := dolo.DefaultClient

	imageProperties := make(map[vangogh_integration.ImageType]string)
	if imagesMetadata.Image != "" {
		imageProperties[vangogh_integration.Image] = imagesMetadata.Image
	}
	if imagesMetadata.VerticalImage != "" {
		imageProperties[vangogh_integration.VerticalImage] = imagesMetadata.VerticalImage
	}
	if imagesMetadata.Hero != "" {
		imageProperties[vangogh_integration.Hero] = imagesMetadata.Hero
	} else if imagesMetadata.Background != "" {
		imageProperties[vangogh_integration.Hero] = imagesMetadata.Background
	}
	if imagesMetadata.Logo != "" {
		imageProperties[vangogh_integration.Logo] = imagesMetadata.Logo
	}
	if imagesMetadata.IconSquare != "" {
		imageProperties[vangogh_integration.IconSquare] = imagesMetadata.IconSquare
	} else if imagesMetadata.Icon != "" {
		imageProperties[vangogh_integration.Icon] = imagesMetadata.Icon
	}

	for ip, imageId := range imageProperties {
		srcUrl, err := data.VangoghUrl(rdx, data.VangoghImagePath, map[string]string{"id": imageId})
		if err != nil {
			return dsgia.EndWithError(err)
		}
		dstFilename := vangogh_integration.SteamGridImageFilename(shortcutId, ip)
		if err := dc.Download(srcUrl, force, nil, absSteamGridPath, dstFilename); err != nil {
			dsgia.Error(err)
		}
	}

	return nil
}

func addNonSteamAppShortcut(
	appId uint32,
	appName, exe, icon, startDir, launchOptions string,
	kvUserShortcuts []*steam_vdf.KeyValues,
	force bool) (bool, error) {

	ansasa := nod.Begin(" adding non-Steam app shortcut for appId %d...", appId)
	defer ansasa.EndWithResult("done")

	kvShortcuts := steam_vdf.GetKevValuesByKey(kvUserShortcuts, "shortcuts")
	if kvShortcuts == nil {
		return false, errors.New("provided shortcuts.vdf is missing shortcuts key")
	}

	if existingShortcut := steam_integration.GetShortcutByAppId(kvShortcuts, appId); existingShortcut == nil || force {

		shortcut := steam_integration.NewShortcut()

		shortcut.AppId = appId
		shortcut.AppName = appName
		shortcut.Exe = exe
		shortcut.StartDir = startDir
		shortcut.LaunchOptions = launchOptions
		shortcut.Icon = icon

		if existingShortcut == nil {

			if err := steam_integration.AppendShortcut(kvShortcuts, shortcut); err != nil {
				return false, err
			}
			ansasa.EndWithResult("appended shortcut")

		} else {
			if err := steam_integration.UpdateShortcut(existingShortcut.Key, kvShortcuts, shortcut); err != nil {
				return false, err
			}
			ansasa.EndWithResult("updated shortcut")

		}
		return true, nil

	} else {

		ansasa.EndWithResult("shortcut already exists (use -force to update)")
		return false, nil

	}
}

func readUserShortcuts(loginUser string) ([]*steam_vdf.KeyValues, error) {

	rusa := nod.Begin(" loading Steam user %s shortcuts.vdf...", loginUser)
	defer rusa.EndWithResult("done")

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return nil, rusa.EndWithError(err)
	}

	absUserShortcutsPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", shortcutsFilename)

	if _, err := os.Stat(absUserShortcutsPath); err != nil {
		return nil, rusa.EndWithError(err)
	}

	return steam_vdf.ParseBinary(absUserShortcutsPath)
}

func getGridIconPath(loginUser string, appId uint32) string {
	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid", fmt.Sprintf("%d_icon.png", appId))
}

func writeUserShortcuts(loginUser string, kvUserShortcuts []*steam_vdf.KeyValues) error {
	wusa := nod.Begin(" writing Steam user %s shortcuts.vdf...", loginUser)
	defer wusa.EndWithResult("done")

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return wusa.EndWithError(err)
	}

	absUserShortcutsPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", shortcutsFilename)

	return steam_vdf.WriteBinary(absUserShortcutsPath, kvUserShortcuts)
}

func getSteamLoginUsers() ([]string, error) {
	gslua := nod.Begin(" getting Steam loginusers.vdf users...")
	defer gslua.EndWithResult("done")

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return nil, gslua.EndWithError(err)
	}

	absLoginUsersPath := filepath.Join(udhd, "Steam", "config", loginUsersFilename)

	if _, err := os.Stat(absLoginUsersPath); err != nil {
		return nil, gslua.EndWithError(err)
	}

	kvLoginUsers, err := steam_vdf.ParseText(absLoginUsersPath)
	if err != nil {
		return nil, gslua.EndWithError(err)
	}

	if users := steam_vdf.GetKevValuesByKey(kvLoginUsers, "users"); users != nil {

		steamIds := make([]string, 0, len(users.Values))

		for _, userId := range users.Values {

			steamId, err := steam_integration.SteamIdFromUserId(userId.Key)
			if err != nil {
				return nil, gslua.EndWithError(err)
			}
			steamIds = append(steamIds, strconv.FormatInt(steamId, 10))
		}

		return steamIds, nil

	}

	return nil, gslua.EndWithError(errors.New("failed to successfully process loginusers.vdf"))
}

func steamStateDirExist() (bool, error) {
	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return false, err
	}

	absSteamStatePath := filepath.Join(udhd, "Steam")

	if _, err := os.Stat(absSteamStatePath); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}
