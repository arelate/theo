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

type listTarget int

const (
	ListTargetUnknown listTarget = iota
	ListTargetAvailableProducts
	ListTargetInstalled
	ListTargetLaunchOptions
	ListTargetSteamShortcuts
	ListTargetTasks
)

func ListHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.UrlIdParameter)

	lt := ListTargetUnknown
	if q.Has(vangogh_integration.UrlAvailableProductsParameter) {
		lt = ListTargetAvailableProducts
	} else if q.Has(vangogh_integration.UrlInstalledParameter) {
		lt = ListTargetInstalled
	} else if q.Has(vangogh_integration.UrlLaunchOptionsParameter) {
		lt = ListTargetLaunchOptions
	} else if q.Has(vangogh_integration.UrlSteamShortcutsParameter) {
		lt = ListTargetSteamShortcuts
	} else if q.Has(vangogh_integration.UrlTasksParameter) {
		lt = ListTargetTasks
	}

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.UrlOperatingSystemParameter) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.UrlOperatingSystemParameter))
	}

	var langCode string
	if q.Has(vangogh_integration.UrlLanguageCodeParameter) {
		langCode = q.Get(vangogh_integration.UrlLanguageCodeParameter)
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
		Origin:          data.UnknownOrigin,
		force:           q.Has(vangogh_integration.UrlForceParameter),
	}

	if q.Has(vangogh_integration.UrlSteamParameter) {
		ii.Origin = data.SteamOrigin
	} else if q.Has(vangogh_integration.UrlEpicGamesParameter) {
		ii.Origin = data.EpicGamesOrigin
	}

	update := q.Has(vangogh_integration.UrlUpdateParameter)

	return List(lt, ii, id, update)
}

func List(lt listTarget,
	installInfo *InstallInfo,
	id string, update bool) error {

	switch lt {
	case ListTargetAvailableProducts:
		return listAvailableProducts(installInfo, update)
	case ListTargetInstalled:
		return listInstalled(installInfo)
	case ListTargetLaunchOptions:
		return listLaunchOptions(id, installInfo)
	case ListTargetSteamShortcuts:
		return listSteamShortcuts()
	case ListTargetTasks:
		if id == "" {
			return errors.New("listing tasks requires product id")
		}

		return listTasks(id, installInfo)
	case ListTargetUnknown:
		return errors.New("you need to specify at least one category to list")
	default:
		return errors.New("unknown list target")
	}
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

	if ii.Origin == data.UnknownOrigin {
		ii.Origin = data.VangoghOrigin
	}

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
		title := fmt.Sprintf(" - %s (%s: %s) OS:%v", ap.Title, ii.Origin, ap.Id, ap.OperatingSystems)
		if len(ap.Dlc) > 0 {
			var dlcs []string
			for dlcId, dlcTitle := range ap.Dlc {
				dlcs = append(dlcs, fmt.Sprintf("%s (%s)", dlcTitle, dlcId))
			}
			title += fmt.Sprintf(" DLC:%s", strings.Join(dlcs, "; "))
		}
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
		vangogh_integration.GogTitleProperty,
		vangogh_integration.SteamTitleProperty,
		vangogh_integration.EgsTitleProperty,
		vangogh_integration.GogBundleNameProperty,
		data.InstallInfoProperty,
		data.InstallDateProperty,
		data.LastRunDateProperty,
		data.TotalPlaytimeMinutesProperty)
	if err != nil {
		return err
	}

	summary := make(map[string][]string)

	installedIds := slices.Collect(rdx.Keys(data.InstallInfoProperty))

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

		var titleLine string

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

			titleLine = fmt.Sprintf("%s: %s", installedInfo.Origin, id)

			if title, terr := data.GetTitleProperty(id, rdx); terr == nil && title != "" {
				titleLine = fmt.Sprintf("%s (%s)", title, titleLine)
				installDir = pathways.Sanitize(title)
			} else if terr != nil {
				return terr
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

			summary[titleLine] = append(summary[titleLine], strings.Join(infoLines, "; "))

			if len(installedInfo.DownloadableContent) > 0 {
				summary[titleLine] = append(summary[titleLine], "- dlc: "+strings.Join(installedInfo.DownloadableContent, ", "))
			}

			if installedDate != "" {
				installStr := "- installed: " + installedDate
				if installDir != "" {
					installStr += "; dir: " + installDir
				}
				summary[titleLine] = append(summary[titleLine], installStr)
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
			summary[titleLine] = append(summary[titleLine], playtimeStr)
		}

	}

	if len(summary) == 0 {
		lia.EndWithResult("found nothing")
	} else {
		lia.EndWithSummary("found the following products:", summary)
	}

	return nil
}

func listLaunchOptions(id string, request *InstallInfo) error {

	lloa := nod.Begin("listing launch options for %s...", id)
	defer lloa.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	installedInfo, err := matchInstalledInfo(id, request, rdx)
	if err != nil {
		return err
	}

	appOsLangCode := data.AppOsLangCode(id, installedInfo.OperatingSystem, installedInfo.LangCode)

	summary := make(map[string][]string)

	launchOptionsProperties := []string{
		data.LaunchOptionsExeProperty,
		data.LaunchOptionsArgProperty,
		data.LaunchOptionsEnvProperty,
	}

	for _, lop := range launchOptionsProperties {
		if values, ok := rdx.GetAllValues(lop, appOsLangCode); ok && len(values) > 0 {
			summary[lop] = values
		}
	}

	if len(summary) > 0 {
		lloa.EndWithSummary("found launch options:", summary)
	} else {
		lloa.EndWithResult("nothing found")
	}

	return nil
}

func listTasks(id string, request *InstallInfo) error {

	lpta := nod.Begin("listing tasks for %s...", id)
	defer lpta.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	installedInfo, err := matchInstalledInfo(id, request, rdx)
	if err != nil {
		return err
	}

	tasksSummary := make(map[string][]string)

	switch installedInfo.Origin {
	case data.VangoghOrigin:
		tasksSummary, err = listGogInfoPlayTasks(id, installedInfo, rdx)
	case data.SteamOrigin:
		tasksSummary, err = listSteamAppInfoTasks(id, rdx, installedInfo.force)
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

func listEpicGamesTasks(appName string) (map[string][]string, error) {
	return nil, nil
}

func listSteamShortcuts() error {
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
		if err = listUserShortcuts(loginUser); err != nil {
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

	kvShortcuts, err := kvUserShortcuts.At("shortcuts")
	if err != nil {
		return err
	}

	shortcutValues := make(map[string][]string)

	for _, shortcut := range kvShortcuts.Values {
		shortcutKey := fmt.Sprintf("shortcut: %s", shortcut.Key)

		for _, kv := range shortcut.Values {
			if slices.Contains(steamShortcutPrintedKeys, kv.Key) && kv.TypedValue != nil {
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
