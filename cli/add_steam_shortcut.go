package cli

import (
	"errors"
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
	runLaunchOptionsTemplate     = "run {id}"
	wineRunLaunchOptionsTemplate = "wine-run {id}"
)

func AddSteamShortcutHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	force := q.Has("force")

	var launchOptionsTemplate string
	switch q.Has("wine") {
	case true:
		launchOptionsTemplate = wineRunLaunchOptionsTemplate
	case false:
		launchOptionsTemplate = runLaunchOptionsTemplate
	}

	return AddSteamShortcut(langCode, launchOptionsTemplate, force, ids...)
}

func AddSteamShortcut(langCode string, launchOptionsTemplate string, force bool, ids ...string) error {
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
		if err := addSteamShortcutsForUser(loginUser, langCode, launchOptionsTemplate, force, ids...); err != nil {
			return assa.EndWithError(err)
		}
	}

	return nil
}

func addSteamShortcutsForUser(loginUser string, langCode string, launchOptionsTemplate string, force bool, ids ...string) error {

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

	for _, id := range ids {

		shortcut, err := createSteamShortcut(loginUser, id, langCode, launchOptionsTemplate, rdx)
		if err != nil {
			return asfua.EndWithError(err)
		}

		if changed, err := addNonSteamAppShortcut(shortcut, kvUserShortcuts, force); err != nil {
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

		if err := downloadSteamGridImages(loginUser, shortcut.AppId, &metadata.Images, rdx, force); err != nil {
			return asfua.EndWithError(err)
		}
	}

	return nil
}

func createSteamShortcut(loginUser, id, langCode string, launchOptionsTemplate string, rdx kevlar.ReadableRedux) (*steam_integration.Shortcut, error) {

	var title string
	if tp, ok := rdx.GetLastVal(data.TitleProperty, id); ok && tp != "" {
		title = tp
	} else {
		return nil, errors.New("add-steam-shortcut: product is missing title")
	}

	shortcutId := steam_integration.ShortcutAppId(title)

	theoExecutable, err := data.TheoExecutable()
	if err != nil {
		return nil, err
	}

	launchOptions := strings.Replace(launchOptionsTemplate, "{id}", id, 1)
	if langCode != "" && langCode != defaultLangCode {
		launchOptions += " -lang-code " + langCode
	}

	shortcut := steam_integration.NewShortcut()

	shortcut.AppId = shortcutId
	shortcut.AppName = title
	shortcut.Exe = theoExecutable
	shortcut.LaunchOptions = launchOptions
	shortcut.Icon = getGridIconPath(loginUser, shortcutId)

	return shortcut, nil
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

func addNonSteamAppShortcut(shortcut *steam_integration.Shortcut, kvUserShortcuts []*steam_vdf.KeyValues, force bool) (bool, error) {

	ansasa := nod.Begin(" adding non-Steam app shortcut for appId %d...", shortcut.AppId)
	defer ansasa.EndWithResult("done")

	kvShortcuts := steam_vdf.GetKevValuesByKey(kvUserShortcuts, "shortcuts")
	if kvShortcuts == nil {
		return false, errors.New("provided shortcuts.vdf is missing shortcuts key")
	}

	if existingShortcut := steam_integration.GetShortcutByAppId(kvShortcuts, shortcut.AppId); existingShortcut == nil || force {

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

	iconFilename := strconv.FormatInt(int64(appId), 10) + "_icon.png"
	return filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid", iconFilename)
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
