package cli

import (
	"errors"
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
	if errors.Is(err, ErrInstallInfoNotFound) {
		ploa.EndWithResult("install info not found")
		return nil
	} else if err != nil {
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
	case "241300":
		// Card City Nights 2
		return fixSteamAppId(id, ii, rdx, false)
	case "3035120":
		// Is This Seat Taken?
		return fixSteamAppId(id, ii, rdx, false)
	default:
		// do nothing
	}

	switch ii.Origin {
	case data.EpicGamesOrigin:
		return presetEpicPortalArg(id, ii)
	default:
		// do nothing
	}

	ploa.EndWithResult("no preset found for " + id)

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
