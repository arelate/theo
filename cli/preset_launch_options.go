package cli

import (
	"errors"
	"net/url"
	"path/filepath"

	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

const (
	defaulEpicUsername   = "epicusername"
	defaultEpicUserId    = "epicuserid"
	defaultEpicSandboxId = "epicsandboxid"
	defaultAuthPassword  = "AUTH_PASSWORD"
)

func PresetLaunchOptionsHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.UrlIdParameter)

	ii := new(InstallInfo{
		OperatingSystem: vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.UrlOperatingSystemParameter)),
		LangCode:        q.Get(vangogh_integration.UrlLanguageCodeParameter),
		EgsAuthPassword: q.Has("epic-games-auth-password"),
	})

	rdx, err := redux.NewWriter(vangogh_integration.AbsReduxDir(), data.AllProperties()...)
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
		return presetBaldursGate3Exe(id, ii, et)
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
		return presetEpicStandaloneMode(id, ii)
	default:
		// do nothing
	}

	ploa.EndWithResult("no preset found for " + id)

	return nil
}

func presetBaldursGate3Exe(id string, ii *InstallInfo, et *execTask) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		dir, fn := filepath.Split(et.exe)
		if fn == "Baldur's Gate 3" {
			et.exe = filepath.Join(dir, "Baldur's Gate 3 GOG")
		}
	default:
		// do nothing
	}

	return LaunchOptions(id, ii, et, false)
}

func presetEpicPortalArg(appName string, ii *InstallInfo) error {
	return LaunchOptions(appName, ii, new(execTask{args: []string{"-EpicPortal"}}), false)
}

func presetEpicStandaloneMode(appName string, ii *InstallInfo) error {

	epicUsername := defaulEpicUsername
	epicUserId := defaultEpicUserId
	epicSandboxId := defaultEpicSandboxId
	epicAuthPassword := defaultAuthPassword

	client, err := egs_integration.GetClient()
	if err != nil {
		return err
	}

	ptr, err := egs_integration.VerifyToken(client)
	if err != nil {
		return err
	}

	epicUsername = ptr.DisplayName
	epicUserId = ptr.AccountId

	gameAssets, err := egs_integration.ReadLocalGameAssets(ii.OperatingSystem)
	if err != nil {
		return err
	}

	for _, ga := range gameAssets {
		if ga.AppName == appName {
			epicSandboxId = ga.Namespace
			break
		}
	}

	if ii.EgsAuthPassword {
		et, err := egs_integration.ExchangeToken(ptr.AccessToken, client)
		if err != nil {
			return err
		}

		epicAuthPassword = et.Code
	}

	return LaunchOptions(appName, ii, new(execTask{args: []string{
		"-AUTH_LOGIN=unused",
		"-AUTH_TYPE=exchangecode",
		"-AUTH_PASSWORD=" + epicAuthPassword,
		"-epicapp=" + appName,
		"-epicenv=Prod",
		"-EpicPortal",
		"-epicusername=" + epicUsername,
		"-epicuserid=" + epicUserId,
		"-epiclocale=en",
		"-epicsandboxid=" + epicSandboxId,
	}}), false)
}
