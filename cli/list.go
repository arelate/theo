package cli

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/steam_vdf"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

var steamShortcutPrintedKeys = []string{
	"appid",
	"appname",
	"icon",
	"Exe",
	"StartDir",
	"LaunchOptions",
}

func ListHandler(u *url.URL) error {

	q := u.Query()

	installed := q.Has("installed")
	playTasks := q.Has("playtasks")
	steamShorts := q.Has("steam-shortcuts")

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	var langCode string
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
	}

	id := q.Get(vangogh_integration.IdProperty)
	allKeyValues := q.Has("all-key-values")

	return List(installed, playTasks, steamShorts, ii, id, allKeyValues)
}

func List(installed, playTasks, steamShortcuts bool,
	installInfo *InstallInfo,
	id string, allKeyValues bool) error {

	if installed || playTasks || steamShortcuts {
		// do nothing
	} else {
		return errors.New("you need to specify at least one category to list")
	}

	if installed {
		if err := listInstalled(installInfo); err != nil {
			return err
		}
	}

	if playTasks {
		if id == "" {
			return errors.New("listing playTasks requires product id")
		}
		if err := listPlayTasks(id, installInfo.LangCode); err != nil {
			return err
		}
	}

	if steamShortcuts {
		if err := listSteamShortcuts(allKeyValues); err != nil {
			return err
		}
	}

	return nil
}

func listInstalled(ii *InstallInfo) error {

	lia := nod.Begin("listing installed products for %s, %s...", ii.OperatingSystem, ii.LangCode)
	defer lia.Done()

	rdx, err := redux.NewReader(data.AbsReduxDir(),
		vangogh_integration.TitleProperty,
		data.InstallInfoProperty,
		data.InstallDateProperty,
		data.LastRunDateProperty,
		data.TotalPlaytimeMinutesProperty)
	if err != nil {
		return err
	}

	summary := make(map[string][]string)

	installedIds := slices.Collect(rdx.Keys(data.InstallInfoProperty))
	installedIds, err = rdx.Sort(installedIds, false, vangogh_integration.TitleProperty)
	if err != nil {
		return err
	}

	for _, id := range installedIds {

		var installedDate string
		if ids, ok := rdx.GetLastVal(data.InstallDateProperty, id); ok && ids != "" {
			if installDate, err := time.Parse(time.RFC3339, ids); err == nil {
				installedDate = installDate.Local().Format(time.DateTime)
			} else {
				return err
			}
		}

		var title string

		filteredIds := make(map[string]any)

		installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id)
		if !ok {
			return errors.New("install info not found for " + id)
		}

		for _, line := range installedInfoLines {

			var installedInfo InstallInfo
			if err = json.UnmarshalRead(strings.NewReader(line), &installedInfo); err != nil {
				return err
			}

			if ii.OperatingSystem != vangogh_integration.AnyOperatingSystem && ii.OperatingSystem != installedInfo.OperatingSystem {
				filteredIds[id] = nil
				continue
			}

			if ii.LangCode != "" && ii.LangCode != installedInfo.LangCode {
				filteredIds[id] = nil
				continue
			}

			title = fmt.Sprintf("%s: %s", ii.Origin, id)
			if tp, sure := rdx.GetLastVal(vangogh_integration.TitleProperty, id); sure && tp != "" {
				title = fmt.Sprintf("%s (%s)", tp, title)
			}

			infoLines := make([]string, 0)

			infoLines = append(infoLines, "os: "+installedInfo.OperatingSystem.String())
			infoLines = append(infoLines, "lang: "+gog_integration.LanguageNativeName(installedInfo.LangCode))

			pfxDt := "type: "
			if len(installedInfo.DownloadTypes) > 1 {
				pfxDt = "types: "
			}
			dts := make([]string, 0, len(installedInfo.DownloadTypes))
			for _, dt := range installedInfo.DownloadTypes {
				dts = append(dts, dt.HumanReadableString())
			}
			infoLines = append(infoLines, pfxDt+strings.Join(dts, ", "))

			if installedInfo.Version != "" {
				infoLines = append(infoLines, "version: "+installedInfo.Version)
			}

			if installedInfo.EstimatedBytes > 0 {
				infoLines = append(infoLines, "size: "+vangogh_integration.FormatBytes(installedInfo.EstimatedBytes))
			}

			summary[title] = append(summary[title], strings.Join(infoLines, "; "))

			if len(installedInfo.DownloadableContent) > 0 {
				summary[title] = append(summary[title], "- dlc: "+strings.Join(installedInfo.DownloadableContent, ", "))
			}

			if installedDate != "" {
				summary[title] = append(summary[title], "- installed: "+installedDate)
			}

		}

		// playtimes

		if _, filtered := filteredIds[id]; filtered {
			continue
		}

		if tpms, sure := rdx.GetLastVal(data.TotalPlaytimeMinutesProperty, id); sure && tpms != "" {
			if tpmi, err := strconv.ParseInt(tpms, 10, 64); err == nil {
				if tpmi > 0 {
					summary[title] = append(summary[title], "- total playtime: "+fmtHoursMinutes(tpmi))
				}
			} else {
				return err
			}
		}

		if lrds, sure := rdx.GetLastVal(data.LastRunDateProperty, id); sure && lrds != "" {
			if lrdt, err := time.Parse(time.RFC3339, lrds); err == nil {
				summary[title] = append(summary[title], "- last run date: "+lrdt.Format(time.DateTime))
			} else {
				return err
			}
		}

	}

	if len(summary) == 0 {
		lia.EndWithResult("found nothing")
	} else {
		lia.EndWithSummary("found the following products:", summary)
	}

	return nil
}

func listPlayTasks(id, langCode string) error {

	lpta := nod.Begin("listing playTasks for %s...", id)
	defer lpta.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	absGogGameInfoPath, err := prefixFindGogGameInfo(id, langCode, rdx)
	if err != nil {
		return err
	}

	gogGameInfo, err := gog_integration.GetGogGameInfo(absGogGameInfoPath)
	if err != nil {
		return err
	}

	playTasksSummary := make(map[string][]string)

	for _, pt := range gogGameInfo.PlayTasks {
		list := make([]string, 0)
		if pt.Arguments != "" {
			list = append(list, "arguments:"+pt.Arguments)
		}
		list = append(list, "category:"+pt.Category)
		if pt.IsPrimary {
			list = append(list, "isPrimary:true")
		}
		if pt.IsHidden {
			list = append(list, "isHidden:true")
		}
		if len(pt.Languages) > 0 {
			list = append(list, "languages:"+strings.Join(pt.Languages, ","))
		}
		if pt.Link != "" {
			list = append(list, "link:"+pt.Link)
		}
		if len(pt.OsBitness) > 0 {
			list = append(list, "osBitness:"+strings.Join(pt.OsBitness, ","))
		}
		if pt.Path != "" {
			list = append(list, "path:"+pt.Path)
		}
		list = append(list, "type:"+pt.Type)
		if pt.WorkingDir != "" {
			list = append(list, "workingDir:"+pt.WorkingDir)
		}

		playTasksSummary["name:"+pt.Name] = list
	}

	lpta.EndWithSummary("found the following playTasks:", playTasksSummary)

	return nil
}

func listSteamShortcuts(allKeyValues bool) error {
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
		if err := listUserShortcuts(loginUser, allKeyValues); err != nil {
			return err
		}
	}

	return nil
}

func listUserShortcuts(loginUser string, allKeyValues bool) error {

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

				var addKeyValue bool
				switch allKeyValues {
				case true:
					addKeyValue = true
				case false:
					addKeyValue = slices.Contains(steamShortcutPrintedKeys, kv.Key) && kv.TypedValue != nil
				}

				if addKeyValue {
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

func fmtHoursMinutes(minutes int64) string {
	hours := minutes / 60
	remainingMinutes := minutes - 60*hours

	var fmtHoursMinutes string
	if remainingMinutes > 0 {
		fmtHoursMinutes = strconv.FormatInt(remainingMinutes, 10) + " min(s)"
	}
	if hours > 0 {
		fmtHoursMinutes = strconv.FormatInt(hours, 10) + "hr(s) " + fmtHoursMinutes
	}

	return fmtHoursMinutes
}
