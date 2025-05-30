package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/redux"
	"maps"
	"slices"
	"strings"
)

type SteamShortcutTarget int

const (
	SteamShortcutTargetUnknown SteamShortcutTarget = iota
	SteamShortcutTargetRun
	SteamShortcutTargetWineRun
	SteamShortcutTargetExe
)

const (
	runLaunchOptionsTemplate     = "run {id} -lang-code {lang-code}"
	wineRunLaunchOptionsTemplate = "wine-run {id} -lang-code {lang-code}"
)

var steamShortcutTargetStrings = map[SteamShortcutTarget]string{
	SteamShortcutTargetUnknown: "unknown",
	SteamShortcutTargetRun:     "run",
	SteamShortcutTargetWineRun: "wine-run",
	SteamShortcutTargetExe:     "exe",
}

func (sst SteamShortcutTarget) String() string {
	if ssts, ok := steamShortcutTargetStrings[sst]; ok {
		return ssts
	}
	return steamShortcutTargetStrings[SteamShortcutTargetUnknown]
}

func ParseSteamShortcutTarget(sst string) SteamShortcutTarget {
	for t, ts := range steamShortcutTargetStrings {
		if ts == sst {
			return t
		}
	}
	return SteamShortcutTargetUnknown
}

func AllSteamShortcutTargets() []string {
	return slices.Collect(maps.Values(steamShortcutTargetStrings))
}

func GetSteamShortcutExeLaunchOptions(id string, langCode string, target SteamShortcutTarget, rdx redux.Readable) (exe string, launchOptions string, err error) {

	theoExecutable, err := data.TheoExecutable()
	if err != nil {
		return "", "", err
	}

	switch target {
	case SteamShortcutTargetRun:
		exe = theoExecutable
		launchOptions = strings.Replace(runLaunchOptionsTemplate, "{id}", id, 1)
		launchOptions = strings.Replace(launchOptions, "{lang-code}", langCode, 1)
	case SteamShortcutTargetWineRun:
		exe = theoExecutable
		launchOptions = strings.Replace(wineRunLaunchOptionsTemplate, "{id}", id, 1)
		launchOptions = strings.Replace(launchOptions, "{lang-code}", langCode, 1)
	case SteamShortcutTargetExe:
		exe, err = findGogGameInfoPrimaryPlaytaskExe(id, langCode, rdx)
		if err != nil {
			return "", "", err
		}

		if exe == "" {
			exe, err = findPrefixGogGamesLnk(id, langCode, rdx)
			if err != nil {
				return "", "", err
			}
		}
	case SteamShortcutTargetUnknown:
		return "", "", errors.New("unknown Steam shortcut target")
	}

	return exe, launchOptions, nil
}
