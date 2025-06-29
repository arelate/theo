package cli

import (
	"errors"
	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func RunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	langCode := "" // installed info language will be used instead of default
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
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

	force := q.Has("force")

	return Run(id, operatingSystem, langCode, et, force)
}

func Run(id string, operatingSystem vangogh_integration.OperatingSystem, langCode string, et *execTask, force bool) error {

	ra := nod.NewProgress("running product %s...", id)
	defer ra.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
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

	currentOs := []vangogh_integration.OperatingSystem{operatingSystem}
	langCodes := []string{langCode}

	vangogh_integration.PrintParams([]string{id}, currentOs, langCodes, nil, true)

	if err = checkProductType(id, rdx, force); err != nil {
		return err
	}

	if err = setLastRunDate(rdx, id); err != nil {
		return err
	}

	return osRun(id, operatingSystem, langCode, rdx, et, force)
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
	}

	return nil
}

func setLastRunDate(rdx redux.Writeable, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return rdx.ReplaceValues(data.LastRunDateProperty, id, now)
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

func osRun(id string, operatingSystem vangogh_integration.OperatingSystem, langCode string, rdx redux.Readable, et *execTask, force bool) error {

	var err error
	if err = osConfirmRunnability(operatingSystem); err != nil {
		return err
	}

	if operatingSystem == vangogh_integration.Windows && data.CurrentOs() != vangogh_integration.Windows {

		var absPrefixDir string
		if absPrefixDir, err = data.GetAbsPrefixDir(id, langCode, rdx); err == nil {
			et.prefix = absPrefixDir
		} else {
			return err
		}

		prefixName, err := data.GetPrefixName(id, rdx)
		if err != nil {
			return err
		}

		langPrefixName := path.Join(prefixName, langCode)

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

		var steamAppId string
		if sai, ok := rdx.GetLastVal(vangogh_integration.SteamAppIdProperty, id); ok {
			steamAppId = sai
		}

		if et.exe != "" {
			return osExec(id, steamAppId, operatingSystem, et, force)
		}
	}

	var absGogGameInfoPath string
	switch et.defaultLauncher {
	case false:
		absGogGameInfoPath, err = osFindGogGameInfo(id, operatingSystem, langCode, rdx)
		if err != nil {
			return err
		}
	case true:
		// do nothing
	}

	switch absGogGameInfoPath {
	case "":
		var absDefaultLauncherPath string
		if absDefaultLauncherPath, err = osFindDefaultLauncher(id, operatingSystem, langCode, rdx); err != nil {
			return err
		}
		if et, err = osExecTaskDefaultLauncher(absDefaultLauncherPath, operatingSystem, et); err != nil {
			return err
		}
	default:
		if et, err = osExecTaskGogGameInfo(absGogGameInfoPath, operatingSystem, et); err != nil {
			return err
		}
	}

	var steamAppId string
	if sai, ok := rdx.GetLastVal(vangogh_integration.SteamAppIdProperty, id); ok {
		steamAppId = sai
	}

	return osExec(id, steamAppId, operatingSystem, et, force)
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

func osExec(gogId, steamAppId string, operatingSystem vangogh_integration.OperatingSystem, et *execTask, force bool) error {

	switch operatingSystem {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		return nixRunExecTask(et)
	case vangogh_integration.Windows:
		currentOs := data.CurrentOs()
		switch currentOs {
		case vangogh_integration.MacOS:
			return macOsWineRunExecTask(et)
		case vangogh_integration.Linux:
			return linuxProtonRunExecTask(gogId, steamAppId, et, force)
		default:
			return currentOs.ErrUnsupported()
		}
	default:
		return operatingSystem.ErrUnsupported()
	}
}
