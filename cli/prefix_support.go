package cli

import (
	"errors"
	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func prefixGetExePath(id, langCode string, rdx redux.Readable) (string, error) {

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return "", err
	}

	if ep, ok := rdx.GetLastVal(data.PrefixExePathProperty, path.Join(prefixName, langCode)); ok && ep != "" {
		absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
		if err != nil {
			return "", err
		}

		return filepath.Join(absPrefixDir, relPrefixDriveCDir, ep), nil
	}

	exePath, err := prefixFindGogGameInfoPrimaryPlayTaskExe(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	if exePath == "" {
		exePath, err = prefixFindGogGamesLnk(id, langCode, rdx)
		if err != nil {
			return "", err
		}
	}

	return exePath, nil
}

func prefixFindGlobFile(id, langCode string, rdx redux.Readable, globPattern string) (string, error) {

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	absPrefixDriveCDir := filepath.Join(absPrefixDir, relPrefixDriveCDir)

	matches, err := filepath.Glob(filepath.Join(absPrefixDriveCDir, globPattern))
	if err != nil {
		return "", err
	}

	filteredMatches := make([]string, 0, len(matches))
	for _, match := range matches {
		if _, filename := filepath.Split(match); strings.HasPrefix(filename, ".") {
			continue
		}
		filteredMatches = append(filteredMatches, match)
	}

	if len(filteredMatches) == 1 {

		if _, err = os.Stat(filteredMatches[0]); err == nil {
			return filteredMatches[0], nil
		} else if os.IsNotExist(err) {
			return "", nil
		} else {
			return "", err
		}

	}

	return "", nil
}

func prefixFindGogGameInstallPath(id, langCode string, rdx redux.Readable) (string, error) {
	fi := nod.Begin(" finding install path...")
	defer fi.Done()

	return prefixFindGlobFile(id, langCode, rdx, gogGameInstallDir)
}

func prefixFindGogGameInfo(id, langCode string, rdx redux.Readable) (string, error) {
	fpggi := nod.Begin(" finding goggame-%s.info...", id)
	defer fpggi.Done()

	return prefixFindGlobFile(id, langCode, rdx, strings.Replace(gogGameInfoGlobTemplate, "{id}", id, -1))
}

func prefixFindGogGamesLnk(id, langCode string, rdx redux.Readable) (string, error) {
	fpl := nod.Begin(" finding .lnk...")
	defer fpl.Done()

	return prefixFindGlobFile(id, langCode, rdx, gogGameLnkGlob)
}

func prefixFindGogGameInfoPrimaryPlayTaskExe(id, langCode string, rdx redux.Readable) (string, error) {

	absGogGameInfoPath, err := prefixFindGogGameInfo(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	gogGameInfo, err := gog_integration.GetGogGameInfo(absGogGameInfoPath)
	if err != nil {
		return "", err
	}

	var relExePath string
	ppt, err := gogGameInfo.GetPlayTask("")
	if err != nil {
		return "", err
	}

	relExePath = ppt.Path

	if relExePath == "" {
		return "", errors.New("cannot determine primary or first playTask for " + id)
	}

	absExeDir, _ := filepath.Split(absGogGameInfoPath)

	return filepath.Join(absExeDir, relExePath), nil
}
