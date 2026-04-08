package cli

import (
	"maps"
	"net/url"
	"os"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

var osEnvDefaults = map[vangogh_integration.OperatingSystem][]string{
	vangogh_integration.MacOS: {
		"CX_GRAPHICS_BACKEND=d3dmetal", // other values: dxmt, dxvk, wined3d
		"WINEMSYNC=1",
		"WINEESYNC=0",
		"ROSETTA_ADVERTISE_AVX=1",
		// "MTL_HUD_ENABLED=1", // not a candidate for default value, adding for reference
	},
}

func LaunchOptionsHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	ii := new(InstallInfo{
		OperatingSystem: vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty)),
		LangCode:        q.Get(vangogh_integration.LanguageCodeProperty),
	})

	et := new(execTask{
		exe: q.Get("exe"),
	})

	if q.Has("env") {
		et.env = strings.Split(q.Get("env"), ",")
	}

	if q.Has("arg") {
		for _, arg := range strings.Split(q.Get("arg"), ",") {
			et.args = append(et.args, strings.TrimPrefix(arg, "\\"))
		}
	}

	reset := q.Has("reset")

	return LaunchOptions(id, ii, et, reset)
}

func LaunchOptions(id string, request *InstallInfo, et *execTask, reset bool) error {

	loa := nod.Begin("setting launch options for %s...", id)
	defer loa.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	ii, err := matchInstalledInfo(id, request, rdx)
	if err != nil {
		return err
	}

	appOsLangCode := data.AppOsLangCode(id, ii.OperatingSystem, ii.LangCode)

	if reset {
		if err = rdx.CutKeys(data.LaunchOptionsExeProperty, appOsLangCode); err != nil {
			return err
		}

		if err = rdx.CutKeys(data.LaunchOptionsArgProperty, appOsLangCode); err != nil {
			return err
		}

		if err = rdx.ReplaceValues(data.LaunchOptionsEnvProperty, appOsLangCode, osEnvDefaults[ii.OperatingSystem]...); err != nil {
			return err
		}
	}

	if et.exe != "" {
		if _, err = os.Stat(et.exe); err != nil {
			return err
		}
		if err = rdx.ReplaceValues(data.LaunchOptionsExeProperty, appOsLangCode, et.exe); err != nil {
			return err
		}
	}

	if len(et.args) > 0 {
		if err = rdx.ReplaceValues(data.LaunchOptionsArgProperty, appOsLangCode, et.args...); err != nil {
			return err
		}
	}

	if len(et.env) > 0 {

		var newEnvs []string

		if curEnv, ok := rdx.GetAllValues(data.LaunchOptionsEnvProperty, appOsLangCode); ok {
			newEnvs = mergeEnv(curEnv, et.env)
		} else {
			newEnvs = et.env
		}

		if err = rdx.ReplaceValues(data.LaunchOptionsEnvProperty, appOsLangCode, newEnvs...); err != nil {
			return err
		}
	}

	return nil
}

func mergeEnv(env1 []string, env2 []string) []string {
	de1, de2 := decodeEnv(env1), decodeEnv(env2)
	maps.Copy(de1, de2)
	return encodeEnv(de1)
}

func decodeEnv(env []string) map[string]string {
	de := make(map[string]string, len(env))
	for _, e := range env {
		if k, v, ok := strings.Cut(e, "="); ok {
			de[k] = v
		}
	}
	return de
}

func encodeEnv(de map[string]string) []string {
	ee := make([]string, 0, len(de))
	for k, v := range de {
		ee = append(ee, k+"="+v)
	}
	return ee
}
