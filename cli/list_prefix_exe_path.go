package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

func ListPrefixExePathHandler(_ *url.URL) error {
	return ListPrefixExePath()
}

func ListPrefixExePath() error {

	lpepa := nod.Begin("listing exe paths for prefixes...")
	defer lpepa.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir,
		data.PrefixExePathProperty,
		vangogh_integration.TitleProperty)
	if err != nil {
		return err
	}

	summary := make(map[string][]string)

	for prefixName := range rdx.Keys(data.PrefixExePathProperty) {
		if exePaths, ok := rdx.GetAllValues(data.PrefixExePathProperty, prefixName); ok {
			for _, ep := range exePaths {
				summary[prefixName] = append(summary[prefixName], ep)
			}
		}
	}

	if len(summary) == 0 {
		lpepa.EndWithResult("found nothing")
	} else {
		lpepa.EndWithSummary("found the following exe paths:", summary)
	}

	return nil
}
