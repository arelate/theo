package cli

import (
	"encoding/json"
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
	verbose := q.Has("verbose")
	force := q.Has("force")

	return WineRun(id, langCode, exePath, env, verbose, force)
}

func WineRun(id string, langCode string, exePath string, env []string, verbose, force bool) error {

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

	if err = resolveProductTitles(rdx, id); err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	prefixEnv, _ := rdx.GetAllValues(data.PrefixEnvProperty, path.Join(prefixName, langCode))
	prefixEnv = mergeEnv(prefixEnv, env)

	if ep, ok := rdx.GetLastVal(data.PrefixExePathProperty, prefixName); ok && ep != "" {
		absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
		if err != nil {
			return err
		}

		exePath = filepath.Join(absPrefixDir, relPrefixDriveCDir, ep)
	}

	if exePath == "" {
		exePath, err = getGogGameInfoExecutable(id, langCode, rdx)
		if err != nil {
			return err
		}
	}

	if exePath == "" {
		exePath, err = getPrefixGogGamesLnk(id, langCode, rdx)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(exePath); err != nil {
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

	return currentOsWineRun(id, langCode, rdx, prefixEnv, verbose, force, exePath)
}

func getPrefixGogGamesLnk(id, langCode string, rdx redux.Readable) (string, error) {

	gpggla := nod.Begin(" locating default .lnk in the install folder for %s...", id)
	defer gpggla.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	absPrefixDriveCDir := filepath.Join(absPrefixDir, relPrefixDriveCDir)

	matches, err := filepath.Glob(filepath.Join(absPrefixDriveCDir, gogInstallationLnkGlob))
	if err != nil {
		return "", err
	}

	if len(matches) == 1 {

		relMatch, err := filepath.Rel(absPrefixDriveCDir, matches[0])
		if err != nil {
			return "", err
		}
		gpggla.EndWithResult("found %s", filepath.Join("C:", relMatch))

		return matches[0], nil
	} else {
		return "", errors.New("cannot locate suitable .lnk in the GOG Games folder")
	}
}

func getPrefixGogGameInfo(id, langCode string, rdx redux.Readable) (string, error) {

	gpggia := nod.Begin("locating goggame-%s.info in the install folder...", id)
	defer gpggia.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	absPrefixDriveCDir := filepath.Join(absPrefixDir, relPrefixDriveCDir)

	gogGameInfoFilename := strings.Replace(gogGameInfoGlob, "{id}", id, -1)
	infoFilePath := filepath.Join(absPrefixDriveCDir, gogGameInfoFilename)

	matches, err := filepath.Glob(infoFilePath)
	if err != nil {
		return "", err
	}

	if len(matches) == 1 {
		relMatch, err := filepath.Rel(absPrefixDriveCDir, matches[0])
		if err != nil {
			return "", err
		}
		gpggia.EndWithResult("found %s", filepath.Join("C:", relMatch))

		return matches[0], nil
	} else {
		return "", errors.New("cannot locate goggame-" + id + ".info in the GOG Games folder")
	}
}

func getGogGameInfoExecutable(id, langCode string, rdx redux.Readable) (string, error) {

	absGogGameInfoPath, err := getPrefixGogGameInfo(id, langCode, rdx)
	if err != nil {
		return "", err
	}

	gogGameInfoFile, err := os.Open(absGogGameInfoPath)
	if err != nil {
		return "", err
	}
	defer gogGameInfoFile.Close()

	var gogGameInfo gog_integration.GogGameInfo

	if err = json.NewDecoder(gogGameInfoFile).Decode(&gogGameInfo); err != nil {
		return "", err
	}

	relExePath := gogGameInfo.GetPrimaryPlayTaskPath()

	absExeDir, _ := filepath.Split(absGogGameInfoPath)

	return filepath.Join(absExeDir, relExePath), nil
}
