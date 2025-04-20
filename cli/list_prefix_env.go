package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

func ListPrefixEnvHandler(_ *url.URL) error {
	return ListPrefixEnv()
}

func ListPrefixEnv() error {

	lpea := nod.Begin("listing environment variables for prefixes...")
	defer lpea.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir,
		data.PrefixEnvProperty,
		vangogh_integration.TitleProperty)
	if err != nil {
		return err
	}

	summary := make(map[string][]string)

	for prefixName := range rdx.Keys(data.PrefixEnvProperty) {

		if env, ok := rdx.GetAllValues(data.PrefixEnvProperty, prefixName); ok {
			for _, e := range env {
				summary[prefixName] = append(summary[prefixName], e)
			}
		}
	}

	if len(summary) == 0 {
		lpea.EndWithResult("found nothing")
	} else {
		lpea.EndWithSummary("found the following env:", summary)
	}

	return nil
}
