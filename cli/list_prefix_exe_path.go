package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"strings"
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
		data.TitleProperty)
	if err != nil {
		return err
	}

	summary := make(map[string][]string)

	for prefixName := range rdx.Keys(data.PrefixExePathProperty) {

		title := ""
		if id, _, ok := strings.Cut(prefixName, "-"); ok {
			if tp, sure := rdx.GetLastVal(data.TitleProperty, id); sure {
				title = tp
			}
		}

		var name string
		if title != "" {
			name = title + " (" + prefixName + ")"
		} else {
			name = prefixName
		}

		if exePaths, ok := rdx.GetAllValues(data.PrefixExePathProperty, prefixName); ok {
			for _, ep := range exePaths {
				summary[name] = append(summary[name], ep)
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
