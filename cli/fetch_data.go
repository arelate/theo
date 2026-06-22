package cli

import (
	"fmt"
	"net/url"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func FetchDataHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.UrlIdParameter)

	ii := new(InstallInfo{
		OperatingSystem: vangogh_integration.AnyOperatingSystem,
		Origin:          data.VangoghOrigin,
	})

	if q.Has(vangogh_integration.UrlOperatingSystemParameter) {
		ii.OperatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.UrlOperatingSystemParameter))
	}

	if q.Has(vangogh_integration.UrlSteamParameter) {
		ii.Origin = data.SteamOrigin
	}

	if q.Has(vangogh_integration.UrlEpicGamesParameter) {
		ii.Origin = data.EpicGamesOrigin
	}

	return FetchData(id, ii)
}

func FetchData(id string, ii *InstallInfo) error {

	fda := nod.Begin("fetching data for %s, %s from %s...", id, ii.OperatingSystem, ii.Origin)
	defer fda.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	originData, err := originGetData(id, ii, rdx, true)
	if err != nil {
		return err
	}

	fmt.Print(originData)

	return nil
}
