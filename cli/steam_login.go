package cli

import "net/url"

func SteamLoginHandler(u *url.URL) error {

	q := u.Query()

	username := q.Get("username")

	return SteamLogin(username)
}

func SteamLogin(username string) error {

	return nil
}
