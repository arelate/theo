package cli

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	useSteamAssets bool
	logoPosition   *logoPosition
}

type logoPosition struct {
	PinnedPosition string `json:"pinnedPosition"`
	WidthPct       int    `json:"nWidthPct"`
	HeightPct      int    `json:"nHeightPct"`
}

func SteamShortcutHandler(u *url.URL) error {

	q := u.Query()

	var add []string
	if q.Has("add") {
		add = strings.Split(q.Get("add"), ",")
	}

	var remove []string
	if q.Has("remove") {
		remove = strings.Split(q.Get("remove"), ",")
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
	}

	sgo := &steamGridOptions{
		useSteamAssets: q.Has("steam-assets"),
		logoPosition:   defaultLogoPosition(),
	}

	if q.Has("pinned-position") {
		sgo.logoPosition.PinnedPosition = q.Get("pinned-position")
	}

	if q.Has("width-pct") {
		wpi, err := strconv.ParseInt(q.Get("width-pct"), 10, 32)
		if err != nil {
			return err
		}
		sgo.logoPosition.WidthPct = int(wpi)
	}

	if q.Has("height-pct") {
		hpi, err := strconv.ParseInt(q.Get("height-pct"), 10, 32)
		if err != nil {
			return err
		}
		sgo.logoPosition.HeightPct = int(hpi)
	}

	updateAllInstalled := q.Has("update-all-installed")
	force := q.Has("force")

	return SteamShortcut(add, remove, updateAllInstalled, ii, sgo, force)

}

func SteamShortcut(add []string, remove []string, updateAllInstalled bool, ii *InstallInfo, sgo *steamGridOptions, force bool) error {

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	if updateAllInstalled {
		for id := range rdx.Keys(data.InstallInfoProperty) {
			add = append(add, id)
		}
	}

	if len(add) == 0 && len(remove) == 0 {
		return errors.New("steam-shortcut requires product ids to add or remove")
	}

	if len(remove) > 0 {
		if err = removeSteamShortcut(rdx, remove...); err != nil {
			return err
		}
	}

	for _, id := range add {
		if err = addSteamShortcut(id, ii.OperatingSystem, ii.LangCode, sgo, rdx, force); err != nil {
			return err
		}
	}

	return nil
}

func addSteamShortcut(id string, operatingSystem vangogh_integration.OperatingSystem, langCode string, sgo *steamGridOptions, rdx redux.Writeable, force bool) error {
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

	if operatingSystem == vangogh_integration.AnyOperatingSystem {
		iios, err := installedInfoOperatingSystem(id, rdx)
		if err != nil {
			return err
		}

		operatingSystem = iios
	}

	if langCode == "" {
		lc, err := installedInfoLangCode(id, operatingSystem, rdx)
		if err != nil {
			return err
		}

		langCode = lc
	}

	if sgo == nil {
		sgo = &steamGridOptions{
			logoPosition: defaultLogoPosition(),
		}
	}

	for _, loginUser := range loginUsers {
		if err = addSteamShortcutsForUser(loginUser, id, operatingSystem, langCode, sgo, rdx, force); err != nil {
			return err
		}
	}

	return nil
}

func addSteamShortcutsForUser(loginUser string,
	id string,
	operatingSystem vangogh_integration.OperatingSystem, langCode string,
	sgo *steamGridOptions,
	rdx redux.Writeable,
	force bool) error {

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

	shortcut, err := createSteamShortcut(loginUser, id, operatingSystem, langCode, rdx)
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

	productDetails, err := getProductDetails(id, rdx, force)
	if err != nil {
		return err
	}

	var steamAppId string
	if sai, ok := rdx.GetLastVal(vangogh_integration.SteamAppIdProperty, id); ok && sai != "" {
		steamAppId = sai
	} else {
		sgo.useSteamAssets = false
	}

	switch sgo.useSteamAssets {
	case true:
		err = fetchSteamGridImages(loginUser, steamAppId, shortcut.AppId, force)
	case false:
		err = downloadGogGridImages(loginUser, shortcut.AppId, &productDetails.Images, rdx, force)
	}

	if err != nil {
		return err
	}

	if err = setLogoPosition(loginUser, shortcut.AppId, sgo.logoPosition); err != nil {
		return err
	}

	return nil
}

func createSteamShortcut(loginUser string, id string, operatingSystem vangogh_integration.OperatingSystem, langCode string, rdx redux.Readable) (*steam_integration.Shortcut, error) {

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

	theoExecutable, err := data.TheoExecutable()
	if err != nil {
		return nil, err
	}

	launchOptions := make([]string, 0, 3)
	launchOptions = append(launchOptions, strings.Replace(runTemplate, "{id}", id, 1))
	launchOptions = append(launchOptions, strings.Replace(osTemplate, "{operating-system}", operatingSystem.String(), 1))
	launchOptions = append(launchOptions, strings.Replace(langCodeTemplate, "{lang-code}", langCode, 1))

	var installedPath string
	installedPath, err = osInstalledPath(id, langCode, operatingSystem, rdx)
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

func downloadGogGridImages(loginUser string, shortcutId uint32, productImages *vangogh_integration.ProductImages, rdx redux.Readable, force bool) error {

	dga := nod.Begin(" downloading GOG.com grid images...")
	defer dga.Done()

	if err := rdx.MustHave(data.ServerConnectionProperties); err != nil {
		return err
	}

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return err
	}

	absSteamGridPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid")
	dc := dolo.DefaultClient

	if token, ok := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerSessionToken); ok && token != "" {
		dc.SetAuthorizationBearer(token)
	}

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
		imageQuery := url.Values{
			"id": {imageId},
		}

		srcUrl, err := data.ServerUrl(data.HttpImagePath, imageQuery, rdx)
		if err != nil {
			return err
		}
		dstFilename := vangogh_integration.SteamGridImageFilename(shortcutId, ip)
		if err = dc.Download(srcUrl, force, nil, absSteamGridPath, dstFilename); err != nil {
			dga.Error(err)
		}
	}

	return nil
}

func fetchSteamGridImages(loginUser string, steamAppId string, shortcutId uint32, force bool) error {

	dsa := nod.Begin(" downloading Steam grid images...")
	defer dsa.Done()

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return err
	}

	absSteamGridPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid")
	dc := dolo.DefaultClient

	imageProperties := make(map[vangogh_integration.ImageType]string)
	imageProperties[vangogh_integration.Image] = "header.jpg"
	imageProperties[vangogh_integration.VerticalImage] = "library_600x900.jpg"
	imageProperties[vangogh_integration.Hero] = "library_hero.jpg"
	imageProperties[vangogh_integration.Logo] = "logo.png"

	for ip, assetFilename := range imageProperties {

		assetUrl := steam_integration.AssetUrl(steamAppId, assetFilename)
		dstFilename := vangogh_integration.SteamGridImageFilename(shortcutId, ip)
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
	relLogoPositionFilename := vangogh_integration.SteamGridLogoPositionFilename(shortcutId)

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

	return json.NewEncoder(lpFile).Encode(&glp)
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

		if err := removeSteamGridImages(loginUser, shortcutId); err != nil {
			return err
		}
	}

	if changed, err := removeNonSteamAppShortcut(kvUserShortcuts, removeShortcutAppIds...); err != nil {
		return err
	} else if changed {
		if err := writeUserShortcuts(loginUser, kvUserShortcuts); err != nil {
			return err
		}
	}

	return nil
}

var steamGridImageTypes = []vangogh_integration.ImageType{
	vangogh_integration.Image,
	vangogh_integration.VerticalImage,
	vangogh_integration.Hero,
	vangogh_integration.Logo,
	vangogh_integration.Icon,
}

func removeSteamGridImages(loginUser string, shortcutId uint32) error {

	rsgia := nod.Begin(" removing Steam Grid images...")
	defer rsgia.Done()

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return err
	}

	absSteamGridPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid")

	for _, it := range steamGridImageTypes {
		dstFilename := vangogh_integration.SteamGridImageFilename(shortcutId, it)
		absDstPath := filepath.Join(absSteamGridPath, dstFilename)
		if _, err := os.Stat(absDstPath); os.IsNotExist(err) {
			continue
		}
		if err := os.Remove(absDstPath); err != nil {
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
