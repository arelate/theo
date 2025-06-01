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

	et := &execTask{
		verbose: q.Has("verbose"),
	}
	if q.Has("env") {
		et.env = strings.Split(q.Get("env"), ",")
	}

	force := q.Has("force")

	return Run(id, langCode, et, force)
}

func Run(id string, langCode string, et *execTask, force bool) error {

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

	if err = checkProductType(id, rdx, force); err != nil {
		return err
	}

	if err = setLastRunDate(rdx, id); err != nil {
		return err
	}

	return currentOsRunApp(id, langCode, rdx, et)
}

func checkProductType(id string, rdx redux.Writeable, force bool) error {

	productDetails, err := GetProductDetails(id, rdx, force)
	if err != nil {
		return err
	}

	switch productDetails.ProductType {
	case vangogh_integration.GameProductType:
		// do nothing, proceed normally
		return nil
	case vangogh_integration.PackProductType:
		return errors.New("cannot run a PACK product, please run included game(s): " +
			strings.Join(productDetails.IncludesGames, ","))
	case vangogh_integration.DlcProductType:
		return errors.New("cannot run a DLC product, please run required game(s): " +
			strings.Join(productDetails.RequiresGames, ","))
	}

	return nil
}

func currentOsRunApp(id, langCode string, rdx redux.Readable, et *execTask) error {

	absBundlePath, err := data.GetAbsBundlePath(id, langCode, data.CurrentOs(), rdx)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absBundlePath); os.IsNotExist(err) {
		return errors.New("cannot find app bundle, please reinstall the app " + id)
	} else if err != nil {
		return err
	}

	if err := currentOsExecute(absBundlePath, et); err != nil {
		return err
	}

	return nil
}

func currentOsExecute(path string, et *execTask) error {
	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		return macOsExecute(path, et)
	case vangogh_integration.Windows:
		return windowsExecute(path, et)
	case vangogh_integration.Linux:
		return linuxExecute(path, et)
	default:
		return errors.New("cannot reveal on unknown operating system")
	}
}

func setLastRunDate(rdx redux.Writeable, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return rdx.ReplaceValues(data.LastRunDateProperty, id, now)
}
