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

	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
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

	availableProducts := q.Has("available-products")
	installed := q.Has("installed")
	tasks := q.Has("tasks")
	steamShortcuts := q.Has("steam-shortcuts")

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
		Origin:          data.VangoghOrigin,
		force:           q.Has("force"),
	}

	if q.Has("steam") {
		ii.Origin = data.SteamOrigin
	} else if q.Has("epic-games") {
		ii.Origin = data.EpicGamesOrigin
	}

	update := q.Has("update")

	id := q.Get(vangogh_integration.IdProperty)
	allShortcutKeys := q.Has("all-shortcut-keys")

	return List(availableProducts, installed, tasks, steamShortcuts, ii, id, allShortcutKeys, update)
}

func List(availableProducts, installed, tasks, steamShortcuts bool,
	installInfo *InstallInfo,
	id string, allShortcutKeys bool, update bool) error {

	if availableProducts || installed || tasks || steamShortcuts {
		// do nothing
	} else {
		return errors.New("you need to specify at least one category to list")
	}

	if availableProducts {
		if err := listAvailableProducts(installInfo, update); err != nil {
			return err
		}
	}

	if installed {
		if err := listInstalled(installInfo); err != nil {
			return err
		}
	}

	if tasks {
		if id == "" {
			return errors.New("listing tasks requires product id")
		}
		if err := listTasks(id, installInfo); err != nil {
			return err
		}
	}

	if steamShortcuts {
		if err := listSteamShortcuts(allShortcutKeys); err != nil {
			return err
		}
	}

	return nil
}

func listAvailableProducts(ii *InstallInfo, update bool) error {

	lapa := nod.Begin("listing available products...")
	defer lapa.Done()

	var availableProducts []vangogh_integration.AvailableProduct
	var err error

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	fetchAvailable := update || ii.force

	switch ii.Origin {
	case data.VangoghOrigin:
		if availableProducts, err = vangoghGetAvailableProducts(fetchAvailable); err != nil {
			return err
		}
	case data.EpicGamesOrigin:
		var osGameAssets map[vangogh_integration.OperatingSystem][]egs_integration.GameAsset
		osGameAssets, err = egsGetGameAssets(fetchAvailable)
		if err != nil {
			return err
		}

		if availableProducts, err = egsGameAssetsAvailableProducts(osGameAssets, ii, rdx); err != nil {
			return err
		}
	default:
		return ii.Origin.ErrUnsupportedOrigin()
	}

	apSummary := make(map[string][]string)

	for _, ap := range availableProducts {
		title := fmt.Sprintf(" - %s (%s: %s) os:%v", ap.Title, ii.Origin, ap.Id, ap.OperatingSystems)
		apSummary[title] = []string{}
	}

	msg := fmt.Sprintf("found %d product(s):", len(availableProducts))
	lapa.EndWithSummary(msg, apSummary)

	return nil
}

func originAvailableProductsKey(origin data.Origin, operatingSystem vangogh_integration.OperatingSystem) string {
	return fmt.Sprintf("%s-%s", origin, operatingSystem)
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
			var installDate time.Time
			if installDate, err = time.Parse(time.RFC3339, ids); err == nil {
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

			var installDir string

			title = fmt.Sprintf("%s: %s", installedInfo.Origin, id)
			if tp, sure := rdx.GetLastVal(vangogh_integration.TitleProperty, id); sure && tp != "" {
				title = fmt.Sprintf("%s (%s)", tp, title)
				installDir = pathways.Sanitize(tp)
			}

			infoLines := make([]string, 0)

			infoLines = append(infoLines, "os: "+installedInfo.OperatingSystem.String())
			infoLines = append(infoLines, "lang: "+gog_integration.LanguageNativeName(installedInfo.LangCode))

			switch installedInfo.Origin {
			case data.VangoghOrigin:
				pfxDt := "type: "
				if len(installedInfo.DownloadTypes) > 1 {
					pfxDt = "types: "
				}
				dts := make([]string, 0, len(installedInfo.DownloadTypes))
				for _, dt := range installedInfo.DownloadTypes {
					dts = append(dts, dt.HumanReadableString())
				}
				infoLines = append(infoLines, pfxDt+strings.Join(dts, ", "))
			default:
				// do nothing
			}

			if installedInfo.Version != "" {
				infoLines = append(infoLines, "version: "+installedInfo.Version)
			}

			if installedInfo.TimeUpdated != "" {
				infoLines = append(infoLines, "updated: "+installedInfo.TimeUpdated)
			}

			if installedInfo.EstimatedBytes > 0 {
				infoLines = append(infoLines, "size: "+vangogh_integration.FormatBytes(installedInfo.EstimatedBytes))
			}

			summary[title] = append(summary[title], strings.Join(infoLines, "; "))

			if len(installedInfo.DownloadableContent) > 0 {
				summary[title] = append(summary[title], "- dlc: "+strings.Join(installedInfo.DownloadableContent, ", "))
			}

			if installedDate != "" {
				installStr := "- installed: " + installedDate
				if installDir != "" {
					installStr += "; dir: " + installDir
				}
				summary[title] = append(summary[title], installStr)
			}
		}

		// playtimes

		if _, filtered := filteredIds[id]; filtered {
			continue
		}

		var playtimeStr string

		if tpms, sure := rdx.GetLastVal(data.TotalPlaytimeMinutesProperty, id); sure && tpms != "" {
			var tpmi int64
			if tpmi, err = strconv.ParseInt(tpms, 10, 64); err == nil {
				if tpmi > 0 {
					playtimeStr = "- total playtime: " + fmtHoursMinutes(tpmi)
				}
			} else {
				return err
			}
		}

		if lrds, sure := rdx.GetLastVal(data.LastRunDateProperty, id); sure && lrds != "" {
			var lrdt time.Time
			if lrdt, err = time.Parse(time.RFC3339, lrds); err == nil {
				lastRunDate := "last run date: " + lrdt.Format(time.DateTime)

				switch playtimeStr {
				case "":
					playtimeStr = "- " + lastRunDate
				default:
					playtimeStr += "; " + lastRunDate
				}
			} else {
				return err
			}
		}

		if playtimeStr != "" {
			summary[title] = append(summary[title], playtimeStr)
		}

	}

	if len(summary) == 0 {
		lia.EndWithResult("found nothing")
	} else {
		lia.EndWithSummary("found the following products:", summary)
	}

	return nil
}

func listTasks(id string, ii *InstallInfo) error {

	lpta := nod.Begin("listing tasks for %s...", id)
	defer lpta.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	installedInfo, err := matchInstalledInfo(id, ii, rdx)
	if err != nil {
		return err
	}

	tasksSummary := make(map[string][]string)

	switch installedInfo.Origin {
	case data.VangoghOrigin:
		tasksSummary, err = listGogInfoPlayTasks(id, installedInfo, rdx)
	case data.SteamOrigin:
		tasksSummary, err = listSteamAppInfoTasks(id, rdx, ii.force)
	default:
		err = installedInfo.Origin.ErrUnsupportedOrigin()
	}

	if err != nil {
		return err
	}

	lpta.EndWithSummary("found the following tasks:", tasksSummary)

	return nil
}

func listGogInfoPlayTasks(gogId string, ii *InstallInfo, rdx redux.Readable) (map[string][]string, error) {

	absGogGameInfoPath, err := prefixFindGogGameInfo(gogId, ii, rdx)
	if err != nil {
		return nil, err
	}

	gogGameInfo, err := gog_integration.GetGogGameInfo(absGogGameInfoPath)
	if err != nil {
		return nil, err
	}

	gogPlayTasks := make(map[string][]string)

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

		gogPlayTasks["title:"+pt.Name] = list
	}

	return gogPlayTasks, nil
}

func listSteamAppInfoTasks(steamAppId string, rdx redux.Writeable, force bool) (map[string][]string, error) {

	appInfoKv, err := steamGetAppInfoKv(steamAppId, rdx, force)
	if err != nil {
		return nil, err
	}

	launchConfigs, err := steamGetLaunchConfigs(steamAppId, appInfoKv)
	if err != nil {
		return nil, err
	}

	steamLaunchConfigTasks := make(map[string][]string)

	for ii, lc := range launchConfigs {

		list := make([]string, 0)

		if lc.Executable != "" {
			list = append(list, "executable:"+lc.Executable)
		}
		if lc.Arguments != "" {
			list = append(list, "arguments:"+lc.Arguments)
		}
		if lc.OsList != "" {
			list = append(list, "oslist:"+lc.OsList)
		}
		if lc.OsArch != "" {
			list = append(list, "osarch:"+lc.OsArch)
		}
		if lc.WorkingDir != "" {
			list = append(list, "workingdir:"+lc.WorkingDir)
		}

		if lc.Description != "" {
			steamLaunchConfigTasks["description:"+lc.Description] = list
		} else {
			steamLaunchConfigTasks["index:"+strconv.Itoa(ii)] = list
		}
	}

	return steamLaunchConfigTasks, nil
}

func listSteamShortcuts(allShortcutKeys bool) error {
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
		if err = listUserShortcuts(loginUser, allShortcutKeys); err != nil {
			return err
		}
	}

	return nil
}

func listUserShortcuts(loginUser string, allShortcutKeys bool) error {

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

	kvShortcuts, err := kvUserShortcuts.At("shortcuts")
	if err != nil {
		return err
	}

	shortcutValues := make(map[string][]string)

	for _, shortcut := range kvShortcuts.Values {
		shortcutKey := fmt.Sprintf("shortcut: %s", shortcut.Key)

		for _, kv := range shortcut.Values {

			var addKeyValue bool
			switch allShortcutKeys {
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

	return nil
}

func fmtHoursMinutes(minutes int64) string {
	hours := minutes / 60
	remainingMinutes := minutes - 60*hours

	var fhm string
	if remainingMinutes > 0 {
		fhm = strconv.FormatInt(remainingMinutes, 10) + " min(s)"
	}
	if hours > 0 {
		fhm = strconv.FormatInt(hours, 10) + "hr(s) " + fhm
	}

	return fhm
}
