package cli

import (
	"fmt"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/boggydigital/nod"
	"net/url"
	"slices"
)

var printedKeys = []string{
	"appid",
	"appname",
	"icon",
	"Exe",
	"StartDir",
	"LaunchOptions",
}

func ListSteamShortcutsHandler(_ *url.URL) error {
	return ListSteamShortcuts()
}

func ListSteamShortcuts() error {
	lssa := nod.Begin("listing Steam shortcuts for all users...")
	defer lssa.Done()

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
	defer lusa.Done()

	kvUserShortcuts, err := readUserShortcuts(loginUser)
	if err != nil {
		return err
	}

	if kvUserShortcuts == nil {
		lusa.EndWithResult("user %s is missing shortcuts file", loginUser)
		return nil
	}

	if kvShortcuts := steam_vdf.GetKevValuesByKey(kvUserShortcuts, "shortcuts"); kvShortcuts != nil {

		shortcutValues := make(map[string][]string)

		for _, shortcut := range kvShortcuts.Values {
			shortcutKey := fmt.Sprintf("shortcut: %s", shortcut.Key)

			for _, kv := range shortcut.Values {
				if slices.Contains(printedKeys, kv.Key) && kv.TypedValue != nil {
					keyValue := fmt.Sprintf("%s: %v", kv.Key, kv.TypedValue)
					shortcutValues[shortcutKey] = append(shortcutValues[shortcutKey], keyValue)
				}
			}
		}

		heading := fmt.Sprintf("Steam user %s shortcuts", loginUser)
		lusa.EndWithSummary(heading, shortcutValues)

	} else {
		lusa.EndWithResult("no shortcuts found")
	}

	return nil
}
