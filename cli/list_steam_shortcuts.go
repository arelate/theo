package cli

import (
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/boggydigital/nod"
	"net/url"
	"slices"
)

func ListSteamShortcutsHandler(_ *url.URL) error {
	return ListSteamShortcuts()
}

func ListSteamShortcuts() error {
	lssa := nod.Begin("listing Steam shortcuts for all users...")
	defer lssa.EndWithResult("done")

	ok, err := steamStateDirExist()
	if err != nil {
		return err
	}

	if !ok {
		lssa.EndWithResult("Steam state dir not found")
		return nil
	}

	loginUsers, err := getSteamLoginUsers()
	if err != nil {
		return err
	}

	for _, loginUser := range loginUsers {
		if err := listUserShortcuts(loginUser); err != nil {
			return err
		}
	}

	return nil
}

func listUserShortcuts(loginUser string) error {

	lusa := nod.Begin("listing shortcuts for %s...", loginUser)
	defer lusa.EndWithResult("done")

	kvUserShortcuts, err := readUserShortcuts(loginUser)
	if err != nil {
		return err
	}

	if kvShortcuts := steam_vdf.GetKevValuesByKey(kvUserShortcuts, "shortcuts"); kvShortcuts != nil {

		for _, shortcut := range kvShortcuts.Values {
			printShortcut(shortcut)
		}

	} else {
		lusa.EndWithResult("no shortcuts found")
	}

	return nil
}

var printedKeys = []string{
	"appid",
	"appname",
	"icon",
	"Exe",
	"StartDir",
	"LaunchOptions",
}

func printShortcut(shortcut *steam_vdf.KeyValues) {
	psa := nod.Begin("shortcut: %s", shortcut.Key)
	defer psa.End()

	for _, kv := range shortcut.Values {
		if slices.Contains(printedKeys, kv.Key) && kv.TypedValue != nil {
			pk := nod.Begin("- %s: %v", kv.Key, kv.TypedValue)
			pk.End()
		}
	}
}
