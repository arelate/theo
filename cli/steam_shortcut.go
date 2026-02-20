package cli

import (
	"encoding/json/v2"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/steam_integration"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	loginUsersFilename = "loginusers.vdf"
	shortcutsFilename  = "shortcuts.vdf"
)

const (
	runTemplate      = "run {id}"
	osTemplate       = "-os {operating-system}"
	langCodeTemplate = "-lang-code {lang-code}"
)

const (
	defaultPinnedPosition = "BottomLeft"
	defaultWidthPct       = 100
	defaultHeightPct      = 100
)

type gridLogoPosition struct {
	Version      int           `json:"nVersion"`
	LogoPosition *logoPosition `json:"logoPosition"`
}

type steamGridOptions struct {
	ids          []string
	name         string
	assets       map[steam_grid.Asset]*url.URL
	logoPosition *logoPosition
	remove       bool
}

type logoPosition struct {
	PinnedPosition string  `json:"pinnedPosition"`
	WidthPct       float64 `json:"nWidthPct"`
	HeightPct      float64 `json:"nHeightPct"`
}

func SteamShortcutHandler(u *url.URL) error {

	q := u.Query()

	ids, err := vangogh_integration.IdsFromUrl(u)
	if err != nil {
		return err
	}

	sgo := &steamGridOptions{
		ids:          ids,
		logoPosition: defaultLogoPosition(),
		remove:       q.Has("remove"),
	}

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	langCode := ""
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
		force:           q.Has("force"),
	}

	if q.Has("header") {
		sgo.assets[steam_grid.Header], err = url.Parse(q.Get("header"))
		if err != nil {
			return err
		}
	}

	if q.Has("capsule") {
		sgo.assets[steam_grid.LibraryCapsule], err = url.Parse(q.Get("capsule"))
		if err != nil {
			return err
		}
	}

	if q.Has("hero") {
		sgo.assets[steam_grid.LibraryHero], err = url.Parse(q.Get("hero"))
		if err != nil {
			return err
		}
	}

	if q.Has("logo") {
		sgo.assets[steam_grid.LibraryLogo], err = url.Parse(q.Get("logo"))
		if err != nil {
			return err
		}
	}

	if q.Has("icon") {
		sgo.assets[steam_grid.Icon], err = url.Parse(q.Get("icon"))
		if err != nil {
			return err
		}
	}

	if q.Has("pinned-position") {
		sgo.logoPosition.PinnedPosition = q.Get("pinned-position")
	}

	if q.Has("width-pct") {
		var wp float64
		if wp, err = strconv.ParseFloat(q.Get("width-pct"), 64); err == nil {
			sgo.logoPosition.WidthPct = wp
		} else {
			return err
		}
	}

	if q.Has("height-pct") {
		var hp float64
		if hp, err = strconv.ParseFloat(q.Get("height-pct"), 64); err == nil {
			sgo.logoPosition.HeightPct = hp
		} else {
			return err
		}
	}

	return SteamShortcut(ii, sgo)

}

func SteamShortcut(ii *InstallInfo, sgo *steamGridOptions) error {

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	if sgo.remove {
		if err = removeSteamShortcut(rdx, sgo.ids...); err != nil {
			return err
		}
	} else {
		for _, id := range sgo.ids {
			if err = addSteamShortcut(id, ii, rdx, sgo); err != nil {
				return err
			}
		}
	}

	return nil
}

func addSteamShortcut(id string, ii *InstallInfo, rdx redux.Writeable, sgo *steamGridOptions) error {
	assa := nod.Begin("adding Steam shortcuts for %s...", id)
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

	if sgo == nil {
		sgo = &steamGridOptions{
			logoPosition: defaultLogoPosition(),
		}
	}

	for _, loginUser := range loginUsers {
		if err = addSteamShortcutsForUser(loginUser, id, ii, rdx, sgo); err != nil {
			return err
		}
	}

	return nil
}

func addSteamShortcutsForUser(loginUser string,
	id string,
	ii *InstallInfo,
	rdx redux.Writeable,
	sgo *steamGridOptions) error {

	asfua := nod.Begin(" adding Steam user %s shortcuts for %s...", loginUser, id)
	defer asfua.Done()

	kvUserShortcuts, err := readUserShortcuts(loginUser)
	if err != nil {
		return err
	}

	if kvUserShortcuts == nil {
		asfua.EndWithResult("user %s is missing shortcuts file", loginUser)
		return nil
	}

	shortcut, err := createSteamShortcut(loginUser, id, ii, rdx, sgo)
	if err != nil {
		return err
	}

	var changed bool
	if changed, err = addNonSteamAppShortcut(shortcut, kvUserShortcuts, ii.force); err != nil {
		return err
	} else if changed {
		if err = writeUserShortcuts(loginUser, kvUserShortcuts); err != nil {
			return err
		}
	}

	if err = fetchSteamGridImages(loginUser, shortcut.AppId, sgo.assets, ii.force); err != nil {
		return err
	}

	if err = setLogoPosition(loginUser, shortcut.AppId, sgo.logoPosition); err != nil {
		return err
	}

	return nil
}

func createSteamShortcut(loginUser string, id string, ii *InstallInfo, rdx redux.Readable, sgo *steamGridOptions) (*steam_integration.Shortcut, error) {

	if err := rdx.MustHave(vangogh_integration.TitleProperty); err != nil {
		return nil, err
	}

	var title string
	if sgo.name != "" {
		title = sgo.name
	} else {
		if tp, ok := rdx.GetLastVal(vangogh_integration.TitleProperty, id); ok && tp != "" {
			title = tp
		} else {
			return nil, errors.New("add-steam-shortcut: product is missing title")
		}
	}

	shortcutId := steam_integration.ShortcutAppId(title)

	theoExecutable, err := data.TheoExecutable()
	if err != nil {
		return nil, err
	}

	launchOptions := make([]string, 0, 3)

	launchOptions = append(launchOptions, strings.Replace(runTemplate, "{id}", id, 1))
	launchOptions = append(launchOptions, strings.Replace(osTemplate, "{operating-system}", ii.OperatingSystem.String(), 1))

	installedPath, err := originOsInstalledPath(id, ii, rdx)
	if err != nil {
		return nil, err
	}

	shortcut := steam_integration.NewShortcut()

	shortcut.AppId = shortcutId
	shortcut.AppName = title
	shortcut.Exe = theoExecutable
	shortcut.LaunchOptions = strings.Join(launchOptions, " ")
	shortcut.StartDir = installedPath
	shortcut.Icon = getGridIconPath(loginUser, shortcutId)

	return shortcut, nil
}

func fetchSteamGridImages(loginUser string, shortcutId uint32, gridAssets map[steam_grid.Asset]*url.URL, force bool) error {

	dsa := nod.Begin(" downloading Steam grid images...")
	defer dsa.Done()

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return err
	}

	absSteamGridPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid")
	dc := dolo.DefaultClient

	for asset, assetUrl := range gridAssets {
		dstFilename := steam_grid.ImageFilename(shortcutId, asset)
		if err = dc.Download(assetUrl, force, nil, absSteamGridPath, dstFilename); err != nil {
			dsa.Error(err)
		}
	}

	return nil
}

func setLogoPosition(loginUser string, shortcutId uint32, lp *logoPosition) error {

	slpa := nod.Begin(" setting Steam Grid logo position...")
	defer slpa.Done()

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return err
	}

	absSteamGridPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid")
	relLogoPositionFilename := steam_grid.LogoPositionFilename(shortcutId)

	absLogoPositionFilename := filepath.Join(absSteamGridPath, relLogoPositionFilename)

	lpFile, err := os.Create(absLogoPositionFilename)
	if err != nil {
		return err
	}

	defer lpFile.Close()

	glp := gridLogoPosition{
		Version:      1,
		LogoPosition: lp,
	}

	return json.MarshalWrite(lpFile, &glp)
}

func addNonSteamAppShortcut(shortcut *steam_integration.Shortcut, kvUserShortcuts steam_vdf.ValveDataFile, force bool) (bool, error) {

	asa := nod.Begin(" adding non-Steam app shortcut for appId %d...", shortcut.AppId)
	defer asa.Done()

	kvShortcuts := steam_vdf.GetKevValuesByKey(kvUserShortcuts, "shortcuts")
	if kvShortcuts == nil {
		return false, errors.New("provided shortcuts.vdf is missing shortcuts key")
	}

	if existingShortcut := steam_integration.GetShortcutByAppId(kvShortcuts, shortcut.AppId); existingShortcut == nil || force {

		if existingShortcut == nil {
			if err := steam_integration.AppendShortcut(kvShortcuts, shortcut); err != nil {
				return false, err
			}

			asa.EndWithResult("appended shortcut")
		} else {
			if err := steam_integration.UpdateShortcut(existingShortcut.Key, kvShortcuts, shortcut); err != nil {
				return false, err
			}
			asa.EndWithResult("updated shortcut")
		}
		return true, nil

	} else {

		asa.EndWithResult("shortcut already exists (use -force to update)")
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

	userShortcutsFile, err := os.Open(absUserShortcutsPath)
	if err != nil {
		return nil, err
	}
	defer userShortcutsFile.Close()

	return steam_vdf.ReadBinary(userShortcutsFile)
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

	return steam_vdf.CreateBinary(absUserShortcutsPath, kvUserShortcuts, steam_vdf.VdfBackupExisting)
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

	loginUsersFile, err := os.Open(absLoginUsersPath)
	if err != nil {
		return nil, err
	}
	defer loginUsersFile.Close()

	kvLoginUsers, err := steam_vdf.ReadText(loginUsersFile)
	if err != nil {
		return nil, err
	}

	if users := steam_vdf.GetKevValuesByKey(kvLoginUsers, "users"); users != nil {

		steamIds := make([]string, 0, len(users.Values))

		for _, userId := range users.Values {

			var steamId int64
			steamId, err = steam_integration.SteamIdFromUserId(userId.Key)
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

	if _, err = os.Stat(absSteamStatePath); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func removeSteamShortcut(rdx redux.Readable, ids ...string) error {
	rssa := nod.Begin("removing Steam shortcuts for %s...", strings.Join(ids, ","))
	defer rssa.Done()

	ok, err := steamStateDirExist()
	if err != nil {
		return err
	}

	if !ok {
		rssa.EndWithResult("Steam state dir not found")
		return nil
	}

	loginUsers, err := getSteamLoginUsers()
	if err != nil {
		return err
	}

	for _, loginUser := range loginUsers {
		if err := removeSteamShortcutsForUser(loginUser, rdx, ids...); err != nil {
			return err
		}
	}

	return nil
}

func removeSteamShortcutsForUser(loginUser string, rdx redux.Readable, ids ...string) error {

	rsfua := nod.Begin(" removing Steam user %s shortcuts for %s...",
		loginUser,
		strings.Join(ids, ","))
	defer rsfua.Done()

	kvUserShortcuts, err := readUserShortcuts(loginUser)
	if err != nil {
		return err
	}

	if len(kvUserShortcuts) == 0 {
		rsfua.EndWithResult("user %s has no shortcuts", loginUser)
		return nil
	}

	if kvShortcuts := steam_vdf.GetKevValuesByKey(kvUserShortcuts, "shortcuts"); kvShortcuts != nil {
		if len(kvShortcuts.Values) == 0 {
			rsfua.EndWithResult("user %s has no shortcuts", loginUser)
			return nil
		}
	}

	removeShortcutAppIds := make([]uint32, 0, len(ids))

	for _, id := range ids {

		var title string
		if tp, ok := rdx.GetLastVal(vangogh_integration.TitleProperty, id); ok && tp != "" {
			title = tp
		} else {
			return errors.New("product is missing title")
		}

		shortcutId := steam_integration.ShortcutAppId(title)

		removeShortcutAppIds = append(removeShortcutAppIds, shortcutId)

		if err = removeSteamGridImages(loginUser, shortcutId); err != nil {
			return err
		}
	}

	var changed bool
	if changed, err = removeNonSteamAppShortcut(kvUserShortcuts, removeShortcutAppIds...); err != nil {
		return err
	} else if changed {
		if err = writeUserShortcuts(loginUser, kvUserShortcuts); err != nil {
			return err
		}
	}

	return nil
}

func removeSteamGridImages(loginUser string, shortcutId uint32) error {

	rsgia := nod.Begin(" removing Steam Grid images...")
	defer rsgia.Done()

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return err
	}

	absSteamGridPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid")

	for _, asset := range steam_grid.ShortcutAssets {
		dstFilename := steam_grid.ImageFilename(shortcutId, asset)
		absDstPath := filepath.Join(absSteamGridPath, dstFilename)
		if _, err = os.Stat(absDstPath); os.IsNotExist(err) {
			continue
		}
		if err = os.Remove(absDstPath); err != nil {
			return err
		}
	}

	return nil
}

func defaultLogoPosition() *logoPosition {
	return &logoPosition{
		PinnedPosition: defaultPinnedPosition,
		WidthPct:       defaultWidthPct,
		HeightPct:      defaultHeightPct,
	}
}

func removeNonSteamAppShortcut(
	kvUserShortcuts []*steam_vdf.KeyValues,
	shortcutsIds ...uint32) (bool, error) {

	shortcutsStrs := make([]string, 0, len(shortcutsIds))
	for _, shortcutId := range shortcutsIds {
		shortcutsStrs = append(shortcutsStrs, strconv.FormatInt(int64(shortcutId), 10))
	}

	rnsasa := nod.Begin(" removing non-Steam app shortcut for appIds: %s...",
		strings.Join(shortcutsStrs, ","))
	defer rnsasa.Done()

	kvShortcuts := steam_vdf.GetKevValuesByKey(kvUserShortcuts, "shortcuts")
	if kvShortcuts == nil {
		return false, errors.New("provided shortcuts.vdf is missing shortcuts key")
	}

	if err := steam_integration.RemoveShortcuts(kvShortcuts, shortcutsIds...); err != nil {
		return false, err
	}

	return true, nil
}
