package cli

import "net/url"

func InstallHandler(u *url.URL) error {
	return Install()
}

func Install() error {
	return nil
}
