package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	linuxStartShFilename = "start.sh"
)

func RunHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	var env []string
	if q.Has("env") {
		env = strings.Split(q.Get("env"), ",")
	}
	verbose := q.Has("verbose")

	return Run(id, langCode, env, verbose)
}

func Run(id string, langCode string, env []string, verbose bool) error {

	ra := nod.NewProgress("running product %s...", id)
	defer ra.Done()

	currentOs := []vangogh_integration.OperatingSystem{data.CurrentOs()}
	langCodes := []string{langCode}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	vangogh_integration.PrintParams([]string{id}, currentOs, langCodes, nil, true)

	if err = setLastRunDate(rdx, id); err != nil {
		return err
	}

	return currentOsRunApp(id, langCode, rdx, env, verbose)
}

func currentOsRunApp(id, langCode string, rdx redux.Readable, env []string, verbose bool) error {

	absBundlePath, err := data.GetAbsBundlePath(id, langCode, data.CurrentOs(), rdx)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absBundlePath); err != nil {
		return err
	}

	if err := currentOsExecute(absBundlePath, env, verbose); err != nil {
		return err
	}

	return nil
}

func currentOsExecute(path string, env []string, verbose bool) error {
	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		return macOsExecute(path, env, verbose)
	case vangogh_integration.Windows:
		return windowsExecute(path, env, verbose)
	case vangogh_integration.Linux:
		return linuxExecute(path, env, verbose)
	default:
		return errors.New("cannot reveal on unknown operating system")
	}
}

func setLastRunDate(rdx redux.Writeable, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return rdx.ReplaceValues(data.LastRunDateProperty, id, now)
}
