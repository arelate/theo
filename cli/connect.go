package cli

import (
	"errors"
	"net/url"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func ConnectHandler(u *url.URL) error {

	q := u.Query()

	urlStr := q.Get(vangogh_integration.UrlUrlParameter)

	username := q.Get(vangogh_integration.UrlUsernameParameter)
	password := q.Get(vangogh_integration.UrlPasswordParameter)

	var origin data.Origin

	if q.Has(vangogh_integration.UrlSteamParameter) {
		origin = data.SteamOrigin
	} else if q.Has(vangogh_integration.UrlEpicGamesParameter) {
		origin = data.EpicGamesOrigin
	} else {
		origin = data.VangoghOrigin
	}

	cookies := q.Get(vangogh_integration.UrlCookiesParameter)

	reset := q.Has(vangogh_integration.UrlResetParameter)

	return Connect(urlStr, username, password, cookies, origin, reset)
}

func Connect(urlStr, username, password, cookies string, origin data.Origin, reset bool) error {

	ca := nod.Begin("setting up theo connection...")
	defer ca.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	switch origin {
	case data.VangoghOrigin:
		return vangoghSetupConnection(urlStr, username, password, rdx, reset)
	case data.SteamOrigin:
		if password != "" {
			return errors.New("steam password will be requested by SteamCMD")
		}
		return steamSetupConnection(username, rdx, reset)
	case data.EpicGamesOrigin:
		return egsSetupConnection(cookies, reset)
	default:
		return origin.ErrUnsupportedOrigin()
	}
}
