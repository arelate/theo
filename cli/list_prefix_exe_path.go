package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"strings"
)

func ListPrefixExePathHandler(_ *url.URL) error {
	return ListPrefixExePath()
}

func ListPrefixExePath() error {

	lpepa := nod.Begin("listing exe paths for prefixes...")
	defer lpepa.EndWithResult("done")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return lpepa.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir,
		data.PrefixExePathProperty,
		data.TitleProperty)
	if err != nil {
		return lpepa.EndWithError(err)
	}

	summary := make(map[string][]string)

	for _, prefixName := range rdx.Keys(data.PrefixExePathProperty) {

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
