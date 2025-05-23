package cli

import (
	"errors"
	"github.com/arelate/southern_light/steam_integration"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
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

func AddSteamShortcutHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	force := q.Has("force")

	target := ParseSteamShortcutTarget(q.Get("target"))

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return AddSteamShortcut(target, langCode, rdx, force, ids...)
}

func AddSteamShortcut(target SteamShortcutTarget, langCode string, rdx redux.Writeable, force bool, ids ...string) error {
	assa := nod.Begin("adding Steam shortcuts for %s...", strings.Join(ids, ","))
	defer assa.Done()

	ok, err := steamStateDirExist()
	if err != nil {
		return err
	}

	if !ok {
		assa.EndWithResult("Steam state dir not found")
		return nil
	}

	loginUsers, err := getSteamLoginUsers()
	if err != nil {
		return err
	}

	for _, loginUser := range loginUsers {
		if err = addSteamShortcutsForUser(loginUser, target, langCode, rdx, force, ids...); err != nil {
			return err
		}
	}

	return nil
}

func addSteamShortcutsForUser(loginUser string, target SteamShortcutTarget, langCode string, rdx redux.Writeable, force bool, ids ...string) error {

	asfua := nod.Begin(" adding Steam user %s shortcuts for %s...",
		loginUser,
		strings.Join(ids, ","))
	defer asfua.Done()

	kvUserShortcuts, err := readUserShortcuts(loginUser)
	if err != nil {
		return err
	}

	if kvUserShortcuts == nil {
		asfua.EndWithResult("user %s is missing shortcuts file", loginUser)
		return nil
	}

	for _, id := range ids {

		shortcut, err := createSteamShortcut(loginUser, target, id, langCode, rdx)
		if err != nil {
			return err
		}

		if changed, err := addNonSteamAppShortcut(shortcut, kvUserShortcuts, force); err != nil {
			return err
		} else if changed {
			if err := writeUserShortcuts(loginUser, kvUserShortcuts); err != nil {
				return err
			}
		}

		productDetails, err := GetProductDetails(id, rdx, force)
		if err != nil {
			return err
		}

		if err := fetchSteamGridImages(loginUser, shortcut.AppId, &productDetails.Images, rdx, force); err != nil {
			return err
		}
	}

	return nil
}

func createSteamShortcut(loginUser string, target SteamShortcutTarget, id, langCode string, rdx redux.Readable) (*steam_integration.Shortcut, error) {

	if err := rdx.MustHave(vangogh_integration.TitleProperty); err != nil {
		return nil, err
	}

	var title string
	if tp, ok := rdx.GetLastVal(vangogh_integration.TitleProperty, id); ok && tp != "" {
		title = tp
	} else {
		return nil, errors.New("add-steam-shortcut: product is missing title")
	}

	shortcutId := steam_integration.ShortcutAppId(title)

	exe, launchOptions, err := GetSteamShortcutExeLaunchOptions(id, langCode, target, rdx)
	if err != nil {
		return nil, err
	}

	// this is only required for WINE targets
	var installPath string
	if target == SteamShortcutTargetWineRun || target == SteamShortcutTargetExe {
		installPath, err = findGogGameInstallPath(id, langCode, rdx)
		if err != nil {
			return nil, err
		}
	}

	shortcut := steam_integration.NewShortcut()

	shortcut.AppId = shortcutId
	shortcut.AppName = title
	shortcut.Exe = exe
	shortcut.LaunchOptions = launchOptions
	shortcut.StartDir = installPath
	shortcut.Icon = getGridIconPath(loginUser, shortcutId)

	return shortcut, nil
}

func fetchSteamGridImages(loginUser string, shortcutId uint32, productImages *vangogh_integration.ProductImages, rdx redux.Readable, force bool) error {

	dsgia := nod.Begin(" downloading Steam Grid images...")
	defer dsgia.Done()

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return err
	}

	absSteamGridPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid")
	dc := dolo.DefaultClient

	imageProperties := make(map[vangogh_integration.ImageType]string)
	if productImages.Image != "" {
		imageProperties[vangogh_integration.Image] = productImages.Image
	}
	if productImages.VerticalImage != "" {
		imageProperties[vangogh_integration.VerticalImage] = productImages.VerticalImage
	}
	if productImages.Hero != "" {
		imageProperties[vangogh_integration.Hero] = productImages.Hero
	} else if productImages.Background != "" {
		imageProperties[vangogh_integration.Hero] = productImages.Background
	}
	if productImages.Logo != "" {
		imageProperties[vangogh_integration.Logo] = productImages.Logo
	}
	if productImages.IconSquare != "" {
		imageProperties[vangogh_integration.IconSquare] = productImages.IconSquare
	} else if productImages.Icon != "" {
		imageProperties[vangogh_integration.Icon] = productImages.Icon
	}

	for ip, imageId := range imageProperties {
		srcUrl, err := data.ServerUrl(rdx, data.ServerImagePath, map[string]string{"id": imageId})
		if err != nil {
			return err
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
	defer ansasa.Done()

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
	defer rusa.Done()

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return nil, err
	}

	absUserShortcutsPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", shortcutsFilename)

	if _, err = os.Stat(absUserShortcutsPath); os.IsNotExist(err) {
		// initialize new Steam shortcuts data if the current users has no shortcuts
		return emptyUserShortcuts(), nil
	} else if err != nil {
		return nil, err
	}

	return steam_vdf.ParseBinary(absUserShortcutsPath)
}

func emptyUserShortcuts() []*steam_vdf.KeyValues {
	var kvShortcuts []*steam_vdf.KeyValues

	kvShortcuts = append(kvShortcuts, &steam_vdf.KeyValues{Key: "shortcuts"})

	return kvShortcuts
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
	defer wusa.Done()

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return err
	}

	absUserShortcutsPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", shortcutsFilename)

	return steam_vdf.WriteBinary(absUserShortcutsPath, kvUserShortcuts)
}

func getSteamLoginUsers() ([]string, error) {
	gslua := nod.Begin(" getting Steam loginusers.vdf users...")
	defer gslua.Done()

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return nil, err
	}

	absLoginUsersPath := filepath.Join(udhd, "Steam", "config", loginUsersFilename)

	if _, err = os.Stat(absLoginUsersPath); err != nil {
		return nil, err
	}

	kvLoginUsers, err := steam_vdf.ParseText(absLoginUsersPath)
	if err != nil {
		return nil, err
	}

	if users := steam_vdf.GetKevValuesByKey(kvLoginUsers, "users"); users != nil {

		steamIds := make([]string, 0, len(users.Values))

		for _, userId := range users.Values {

			steamId, err := steam_integration.SteamIdFromUserId(userId.Key)
			if err != nil {
				return nil, err
			}
			steamIds = append(steamIds, strconv.FormatInt(steamId, 10))
		}

		return steamIds, nil

	}

	return nil, errors.New("failed to successfully process loginusers.vdf")
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
