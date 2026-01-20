package cli

import (
	"errors"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func RunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

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
		force:           q.Has("force"),
	}

	et := &execTask{
		workDir:         q.Get("work-dir"),
		verbose:         q.Has("verbose"),
		playTask:        q.Get("playtask"),
		defaultLauncher: q.Has("default-launcher"),
	}

	if q.Has("env") {
		et.env = strings.Split(q.Get("env"), ",")
	}

	if q.Has("arg") {
		et.args = strings.Split(q.Get("arg"), ",")
	}

	if q.Has("proton-options") {
		et.protonOptions = strings.Split(q.Get("proton-options"), ",")
	}

	return Run(id, ii, et)
}

func Run(id string, ii *InstallInfo, et *execTask) error {

	playSessionStart := time.Now()

	ra := nod.NewProgress("running product %s...", id)
	defer ra.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)

	if err != nil {
		return err
	}

	if err = resolveInstallInfo(id, ii, nil, rdx, installedOperatingSystem, installedLangCode); err != nil {
		return err
	}

	printInstallInfoParams(ii, true, id)

	if err = checkProductType(id, rdx, ii.force); err != nil {
		return err
	}

	if err = setLastRunDate(rdx, id); err != nil {
		return err
	}

	if err = osRun(id, ii, rdx, et); err != nil {
		return err
	}

	playSessionDuration := time.Since(playSessionStart)

	if err = recordPlaytime(rdx, id, playSessionDuration); err != nil {
		return err
	}

	return updateTotalPlaytime(rdx, id)
}

func checkProductType(id string, rdx redux.Writeable, force bool) error {

	productDetails, err := getProductDetails(id, rdx, force)
	if err != nil {
		return err
	}

	switch productDetails.ProductType {
	case vangogh_integration.GameProductType:
		// do nothing, proceed normally
		return nil
	case vangogh_integration.PackProductType:
		return errors.New("cannot run a PACK product, please run included game(s): " +
			strings.Join(productDetails.IncludesGames, ","))
	case vangogh_integration.DlcProductType:
		return errors.New("cannot run a DLC product, please run required game(s): " +
			strings.Join(productDetails.RequiresGames, ","))
	default:
		return errors.New("unsupported product type: " + productDetails.ProductType)
	}
}

func setLastRunDate(rdx redux.Writeable, id string) error {

	if err := rdx.MustHave(data.LastRunDateProperty); err != nil {
		return err
	}

	fmtUtcNow := time.Now().UTC().Format(time.RFC3339)
	return rdx.ReplaceValues(data.LastRunDateProperty, id, fmtUtcNow)
}

func recordPlaytime(rdx redux.Writeable, id string, dur time.Duration) error {

	if err := rdx.MustHave(data.PlaytimeMinutesProperty); err != nil {
		return err
	}

	// this will lose some seconds precision
	fmtDur := strconv.FormatInt(int64(dur.Minutes()), 10)

	return rdx.AddValues(data.PlaytimeMinutesProperty, id, fmtDur)
}

func updateTotalPlaytime(rdx redux.Writeable, id string) error {
	if err := rdx.MustHave(data.PlaytimeMinutesProperty, data.TotalPlaytimeMinutesProperty); err != nil {
		return err
	}

	var totalPlaytimeMinutes int64
	if tpms, ok := rdx.GetAllValues(data.PlaytimeMinutesProperty, id); ok && len(tpms) > 0 {
		for _, mins := range tpms {
			if mini, err := strconv.ParseInt(mins, 10, 64); err == nil {
				totalPlaytimeMinutes += mini
			} else {
				return err
			}
		}
	}

	if totalPlaytimeMinutes > 0 {
		return rdx.ReplaceValues(data.TotalPlaytimeMinutesProperty, id, strconv.FormatInt(totalPlaytimeMinutes, 10))
	} else {
		return nil
	}
}

func osConfirmRunnability(operatingSystem vangogh_integration.OperatingSystem) error {
	if operatingSystem == vangogh_integration.MacOS && data.CurrentOs() != vangogh_integration.MacOS {
		return errors.New("running macOS versions is only supported on macOS")
	}
	if operatingSystem == vangogh_integration.Linux && data.CurrentOs() != vangogh_integration.Linux {
		return errors.New("running Linux versions is only supported on Linux")
	}
	return nil
}

func osRun(id string, ii *InstallInfo, rdx redux.Readable, et *execTask) error {

	var err error
	if err = osConfirmRunnability(ii.OperatingSystem); err != nil {
		return err
	}

	if ii.OperatingSystem == vangogh_integration.Windows && data.CurrentOs() != vangogh_integration.Windows {

		var absPrefixDir string
		if absPrefixDir, err = data.AbsPrefixDir(id, rdx); err == nil {
			et.prefix = absPrefixDir
		} else {
			return err
		}

		var prefixName string
		prefixName, err = data.GetPrefixName(id, rdx)
		if err != nil {
			return err
		}

		langPrefixName := path.Join(prefixName, ii.LangCode)

		if env, ok := rdx.GetAllValues(data.PrefixEnvProperty, langPrefixName); ok {
			et.env = mergeEnv(et.env, env)
		}

		if exe, ok := rdx.GetLastVal(data.PrefixExeProperty, langPrefixName); ok {

			absExePath := filepath.Join(absPrefixDir, exe)
			if _, err = os.Stat(absExePath); err == nil {
				et.name = exe
				et.exe = absExePath
			}

		}

		if arg, ok := rdx.GetAllValues(data.PrefixArgProperty, langPrefixName); ok {
			et.args = append(et.args, arg...)
		}

		if et.exe != "" {
			return osExec(id, ii.OperatingSystem, et, rdx, ii.force)
		}
	}

	var absGogGameInfoPath string
	switch et.defaultLauncher {
	case false:
		absGogGameInfoPath, err = osFindGogGameInfo(id, ii.OperatingSystem, ii.LangCode, rdx)
		if err != nil {
			return err
		}
	case true:
		// do nothing
	}

	switch absGogGameInfoPath {
	case "":
		var absDefaultLauncherPath string
		if absDefaultLauncherPath, err = osFindDefaultLauncher(id, ii.OperatingSystem, ii.LangCode, rdx); err != nil {
			return err
		}
		if et, err = osExecTaskDefaultLauncher(absDefaultLauncherPath, ii.OperatingSystem, et); err != nil {
			return err
		}
	default:
		if et, err = osExecTaskGogGameInfo(absGogGameInfoPath, ii.OperatingSystem, et); err != nil {
			return err
		}
	}

	return osExec(id, ii.OperatingSystem, et, rdx, ii.force)
}

func osFindGogGameInfo(id string, operatingSystem vangogh_integration.OperatingSystem, langCode string, rdx redux.Readable) (string, error) {

	var gogGameInfoPath string
	var err error

	switch operatingSystem {
	case vangogh_integration.MacOS:
		gogGameInfoPath, err = macOsFindGogGameInfo(id, langCode, rdx)
	case vangogh_integration.Linux:
		gogGameInfoPath, err = linuxFindGogGameInfo(id, langCode, rdx)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			gogGameInfoPath, err = prefixFindGogGameInfo(id, langCode, rdx)
		case vangogh_integration.Windows:
			gogGameInfoPath, err = windowsFindGogGameInfo(id, langCode, rdx)
		default:
			return "", currentOs.ErrUnsupported()
		}
	default:
		return "", operatingSystem.ErrUnsupported()
	}

	if err != nil {
		return "", err
	}

	return gogGameInfoPath, nil
}

func osExecTaskGogGameInfo(absGogGameInfoPath string, operatingSystem vangogh_integration.OperatingSystem, et *execTask) (*execTask, error) {

	_, gogGameInfoFilename := filepath.Split(absGogGameInfoPath)

	eggia := nod.Begin(" running %s...", gogGameInfoFilename)
	defer eggia.Done()

	gogGameInfo, err := gog_integration.GetGogGameInfo(absGogGameInfoPath)
	if err != nil {
		return nil, err
	}

	switch operatingSystem {
	case vangogh_integration.MacOS:
		return macOsExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
	case vangogh_integration.Linux:
		return linuxExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			return macOsExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
		case vangogh_integration.Linux:
			return linuxExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
		case vangogh_integration.Windows:
			return windowsExecTaskGogGameInfo(absGogGameInfoPath, gogGameInfo, et)
		default:
			return nil, currentOs.ErrUnsupported()
		}
	default:
		return nil, operatingSystem.ErrUnsupported()
	}
}

func osFindDefaultLauncher(id string, operatingSystem vangogh_integration.OperatingSystem, langCode string, rdx redux.Readable) (string, error) {

	var defaultLauncherPath string
	var err error

	switch operatingSystem {
	case vangogh_integration.MacOS:
		defaultLauncherPath, err = macOsFindBundleApp(id, langCode, rdx)
	case vangogh_integration.Linux:
		defaultLauncherPath, err = linuxFindStartSh(id, langCode, rdx)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			defaultLauncherPath, err = prefixFindGogGamesLnk(id, langCode, rdx)
		case vangogh_integration.Windows:
			defaultLauncherPath, err = windowsFindGogGamesLnk(id, langCode, rdx)
		default:
			return "", currentOs.ErrUnsupported()
		}
	default:
		return "", operatingSystem.ErrUnsupported()
	}

	if err != nil {
		return "", err
	}

	return defaultLauncherPath, nil
}

func osExecTaskDefaultLauncher(absDefaultLauncherPath string, operatingSystem vangogh_integration.OperatingSystem, et *execTask) (*execTask, error) {

	_, defaultLauncherFilename := filepath.Split(absDefaultLauncherPath)

	et.name = defaultLauncherFilename

	eggia := nod.Begin(" running %s...", defaultLauncherFilename)
	defer eggia.Done()

	switch operatingSystem {
	case vangogh_integration.MacOS:
		return macOsExecTaskBundleApp(absDefaultLauncherPath, et)
	case vangogh_integration.Linux:
		return linuxExecTaskStartSh(absDefaultLauncherPath, et)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			et.exe = absDefaultLauncherPath
		case vangogh_integration.Windows:
			return windowsExecTaskLnk(absDefaultLauncherPath, et)
		default:
			return nil, currentOs.ErrUnsupported()
		}
	default:
		return nil, operatingSystem.ErrUnsupported()
	}

	return et, nil
}

func osExec(id string, operatingSystem vangogh_integration.OperatingSystem, et *execTask, rdx redux.Readable, force bool) error {

	switch operatingSystem {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		return nixRunExecTask(et)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			return macOsWineRunExecTask(et, rdx)
		case vangogh_integration.Linux:
			return linuxProtonRunExecTask(id, et, rdx, force)
		default:
			return currentOs.ErrUnsupported()
		}
	default:
		return operatingSystem.ErrUnsupported()
	}
}
