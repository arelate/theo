package cli

import (
	"net/url"
	"path/filepath"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func PresetLaunchOptionsHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	ii := new(InstallInfo{
		OperatingSystem: vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty)),
		LangCode:        q.Get(vangogh_integration.LanguageCodeProperty),
	})

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	return PresetLaunchOptions(id, ii, rdx)
}

func PresetLaunchOptions(id string, request *InstallInfo, rdx redux.Writeable) error {

	ploa := nod.Begin("presetting launch options...")
	defer ploa.Done()

	ii, err := matchInstalledInfo(id, request, rdx)
	if err != nil {
		return err
	}

	originData, err := originGetData(id, ii, rdx, false)
	if err != nil {
		return err
	}

	et := new(execTask)
	if et, err = originGetExecTask(id, ii, originData, et, rdx); err != nil {
		return err
	}

	switch id {
	case "1456460669":
		// Baldur's Gate 3
		return presetBaldursGate3Exe(ii, et)
	case "be23672deb69402781cd47cc2919caf4":
		// Marvel's Spider-Man Remastered
		return presetEpicPortalArg(id, ii)
	case "cd231060e6744ffb97684767b07d2b77":
		// Ghost of Tsushima DIRECTOR'S CUT
		return presetEpicPortalArg(id, ii)
	case "eb2c955d6a494dfb9b7e09e0145668a1":
		// Marvel’s Spider-Man: Miles Morales
		return presetEpicPortalArg(id, ii)
	case "0c6db5941ac8420aabbd8510e94e7d9a":
		// Marvel’s Spider-Man 2
		return presetEpicPortalArg(id, ii)
	case "2efe99166b8847e9bcd80c571b05e1b6":
		// Horizon Forbidden West™ Complete Edition
		return presetEpicPortalArg(id, ii)
	case "Kiwi":
		// Tetris® Effect: Connected
		return presetEpicPortalArg(id, ii)
	case "241300":
		// Card City Nights 2
		return fixSteamAppId(id, ii, rdx, false)
	default:
		ploa.EndWithResult("no preset found for " + id)
	}

	return nil
}

func presetBaldursGate3Exe(ii *InstallInfo, et *execTask) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		dir, fn := filepath.Split(et.exe)
		if fn == "Baldur's Gate 3" {
			et.exe = filepath.Join(dir, "Baldur's Gate 3 GOG")
		}
	default:
		// do nothing
	}

	return nil
}

func presetEpicPortalArg(appName string, ii *InstallInfo) error {
	return LaunchOptions(appName, ii, new(execTask{args: []string{"-EpicPortal"}}), false)
}
