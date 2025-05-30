package cli

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

const (
	gogGameInstallDir      = gogGamesDir + "/*"
	gogInstallationLnkGlob = gogGamesDir + "/*/*.lnk"
	gogGameInfoGlob        = gogGamesDir + "/*/goggame-{id}.info"
)

func WineRunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	env := make([]string, 0)
	if q.Has("env") {
		env = strings.Split(q.Get("env"), ",")
	}
	exePath := q.Get("exe-path")
	installPath := q.Get("install-path")
	verbose := q.Has("verbose")
	force := q.Has("force")

	return WineRun(id, langCode, exePath, installPath, env, verbose, force)
}

func WineRun(id string, langCode string, exePath, installPath string, env []string, verbose, force bool) error {

	wra := nod.Begin("running %s version with WINE...", vangogh_integration.Windows)
	defer wra.Done()

	vangogh_integration.PrintParams([]string{id},
		[]vangogh_integration.OperatingSystem{vangogh_integration.Windows},
		[]string{langCode},
		nil,
		false)

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	if err = checkProductType(id, rdx, force); err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	prefixEnv, _ := rdx.GetAllValues(data.PrefixEnvProperty, path.Join(prefixName, langCode))
	prefixEnv = mergeEnv(prefixEnv, env)

	if exePath == "" {
		exePath, err = getExePath(id, langCode, rdx)
		if err != nil {
			return err
		}
	}

	if installPath == "" {
		installPath, err = findGogGameInstallPath(id, langCode, rdx)
		if err != nil {
			return err
		}
	}

	if _, err = os.Stat(exePath); err != nil {
		return err
	}

	if err = setLastRunDate(rdx, id); err != nil {
		return err
	}

	var currentOsWineRun wineRunFunc

	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		currentOsWineRun = macOsWineRun
	case vangogh_integration.Linux:
		currentOsWineRun = linuxProtonRun
	default:
		return errors.New("wine-run: unsupported operating system")
	}

	return currentOsWineRun(id, langCode, rdx, prefixEnv, verbose, force, exePath, installPath)
}

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

	exePath, err := findGogGameInfoPrimaryPlaytaskExe(id, langCode, rdx)
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

func findGogGameInfoPath(id, langCode string, rdx redux.Readable) (string, error) {

	gogGameInfoFilename := strings.Replace(gogGameInfoGlob, "{id}", id, -1)
	absGogGameInfoPath, err := findPrefixFile(id, langCode, rdx, gogGameInfoFilename, ".info file")
	if err != nil {
		return "", err
	}

	return absGogGameInfoPath, nil
}

func getGogGameInfo(absGogGameInfoPath string) (*gog_integration.GogGameInfo, error) {

	gogGameInfoFile, err := os.Open(absGogGameInfoPath)
	if err != nil {
		return nil, err
	}
	defer gogGameInfoFile.Close()

	var gogGameInfo gog_integration.GogGameInfo
	if err = json.NewDecoder(gogGameInfoFile).Decode(&gogGameInfo); err != nil {
		return nil, err
	}

	return &gogGameInfo, nil
}

func findGogGameInfoPrimaryPlaytaskExe(id, langCode string, rdx redux.Readable) (string, error) {

	absGogGameInfoPath, err := findGogGameInfoPath(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	gogGameInfo, err := getGogGameInfo(absGogGameInfoPath)
	if err != nil {
		return "", err
	}

	relExePath := gogGameInfo.GetPrimaryPlayTaskPath()

	absExeDir, _ := filepath.Split(absGogGameInfoPath)

	return filepath.Join(absExeDir, relExePath), nil
}

func findGogGameInstallPath(id, langCode string, rdx redux.Readable) (string, error) {
	return findPrefixFile(id, langCode, rdx, gogGameInstallDir, "install path")
}
