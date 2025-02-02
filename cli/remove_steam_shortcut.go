package cli

import (
	"errors"
	"github.com/arelate/southern_light/steam_integration"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func RemoveSteamShortcutHandler(u *url.URL) error {
	ids := Ids(u)
	return RemoveSteamShortcut(ids...)
}

func RemoveSteamShortcut(ids ...string) error {
	rssa := nod.Begin("removing Steam shortcuts for %s...", strings.Join(ids, ","))
	defer rssa.EndWithResult("done")

	ok, err := steamStateDirExist()
	if err != nil {
		return rssa.EndWithError(err)
	}

	if !ok {
		rssa.EndWithResult("Steam state dir not found")
		return nil
	}

	loginUsers, err := getSteamLoginUsers()
	if err != nil {
		return rssa.EndWithError(err)
	}

	for _, loginUser := range loginUsers {
		if err := removeSteamShortcutsForUser(loginUser, ids...); err != nil {
			return rssa.EndWithError(err)
		}
	}

	return nil
}

func removeSteamShortcutsForUser(loginUser string, ids ...string) error {

	rsfua := nod.Begin(" removing Steam user %s shortcuts for %s...",
		loginUser,
		strings.Join(ids, ","))
	defer rsfua.EndWithResult("done")

	kvUserShortcuts, err := readUserShortcuts(loginUser)
	if err != nil {
		return rsfua.EndWithError(err)
	}

	if kvShortcuts := steam_vdf.GetKevValuesByKey(kvUserShortcuts, "shortcuts"); kvShortcuts != nil {
		if len(kvShortcuts.Values) == 0 {
			rsfua.EndWithResult("%s has no shortcuts", loginUser)
			return nil
		}
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return rsfua.EndWithError(err)
	}

	rdx, err := redux.NewWriter(reduxDir, data.TitleProperty, data.BundleNameProperty, data.SetupProperties)
	if err != nil {
		return rsfua.EndWithError(err)
	}

	removeShortcutAppIds := make([]uint32, 0, len(ids))

	for _, id := range ids {

		var title string
		if tp, ok := rdx.GetLastVal(data.TitleProperty, id); ok && tp != "" {
			title = tp
		} else {
			return rsfua.EndWithError(errors.New("product is missing title"))
		}

		shortcutId := steam_integration.ShortcutAppId(title)

		removeShortcutAppIds = append(removeShortcutAppIds, shortcutId)

		if err := removeSteamGridImages(loginUser, shortcutId); err != nil {
			return rsfua.EndWithError(err)
		}
	}

	if changed, err := removeNonSteamAppShortcut(kvUserShortcuts, removeShortcutAppIds...); err != nil {
		return rsfua.EndWithError(err)
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
	defer rsgia.EndWithResult("done")

	udhd, err := data.UserDataHomeDir()
	if err != nil {
		return rsgia.EndWithError(err)
	}

	absSteamGridPath := filepath.Join(udhd, "Steam", "userdata", loginUser, "config", "grid")

	for _, it := range steamGridImageTypes {
		dstFilename := vangogh_integration.SteamGridImageFilename(shortcutId, it)
		absDstPath := filepath.Join(absSteamGridPath, dstFilename)
		if _, err := os.Stat(absDstPath); os.IsNotExist(err) {
			continue
		}
		if err := os.Remove(absDstPath); err != nil {
			return rsgia.EndWithError(err)
		}
	}

	return nil
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
	defer rnsasa.EndWithResult("done")

	kvShortcuts := steam_vdf.GetKevValuesByKey(kvUserShortcuts, "shortcuts")
	if kvShortcuts == nil {
		return false, errors.New("provided shortcuts.vdf is missing shortcuts key")
	}

	if err := steam_integration.RemoveShortcuts(kvShortcuts, shortcutsIds...); err != nil {
		return false, err
	}

	return true, nil
}
