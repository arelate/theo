package cli

import (
	"errors"
	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
	"path"
	"path/filepath"
	"strings"
)

func getExePath(id, langCode string, rdx redux.Readable) (string, error) {

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

	exePath, err := findPrefixGogGameInfoPrimaryPlayTaskExe(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	if exePath == "" {
		exePath, err = findPrefixGogGamesLnk(id, langCode, rdx)
		if err != nil {
			return "", err
		}
	}

	return exePath, nil
}

func findPrefixGogGamesLnk(id, langCode string, rdx redux.Readable) (string, error) {
	return findPrefixFile(id, langCode, rdx, gogInstallationLnkGlob, "default .lnk")
}

func findPrefixFile(id, langCode string, rdx redux.Readable, globPattern string, msg string) (string, error) {
	fpfa := nod.Begin(" locating %s...", msg)
	defer fpfa.Done()

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
		relMatch, err := filepath.Rel(absPrefixDriveCDir, filteredMatches[0])
		if err != nil {
			return "", err
		}
		fpfa.EndWithResult("found %s", filepath.Join("C:", relMatch))

		return filteredMatches[0], nil
	} else {
		return "", errors.New("cannot locate suitable file in the GOG Games folder")
	}
}

func findPrefixGogGameInfoPath(id, langCode string, rdx redux.Readable) (string, error) {

	gogGameInfoFilename := strings.Replace(gogGameInfoGlob, "{id}", id, -1)
	absGogGameInfoPath, err := findPrefixFile(id, langCode, rdx, gogGameInfoFilename, ".info file")
	if err != nil {
		return "", err
	}

	return absGogGameInfoPath, nil
}

func findPrefixGogGameInfoPrimaryPlayTaskExe(id, langCode string, rdx redux.Readable) (string, error) {

	absGogGameInfoPath, err := findPrefixGogGameInfoPath(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	gogGameInfo, err := gog_integration.GetGogGameInfo(absGogGameInfoPath)
	if err != nil {
		return "", err
	}

	var relExePath string
	if ppt := gogGameInfo.PrimaryPlayTask(); ppt != nil {
		relExePath = ppt.Path
	} else if len(gogGameInfo.PlayTasks) > 0 {
		relExePath = gogGameInfo.PlayTasks[0].Path
	}

	if relExePath == "" {
		return "", errors.New("cannot determine primary or first playTask for " + id)
	}

	absExeDir, _ := filepath.Split(absGogGameInfoPath)

	return filepath.Join(absExeDir, relExePath), nil
}

func findGogGameInstallPath(id, langCode string, rdx redux.Readable) (string, error) {
	return findPrefixFile(id, langCode, rdx, gogGameInstallDir, "install path")
}
