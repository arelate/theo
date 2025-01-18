package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"strings"
)

func ListPrefixEnvHandler(_ *url.URL) error {
	return ListPrefixEnv()
}

func ListPrefixEnv() error {

	lpea := nod.Begin("listing environment variables for prefixes...")
	defer lpea.EndWithResult("done")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return lpea.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir,
		data.PrefixEnvProperty,
		data.TitleProperty)
	if err != nil {
		return lpea.EndWithError(err)
	}

	summary := make(map[string][]string)

	for _, prefixName := range rdx.Keys(data.PrefixEnvProperty) {

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

		if env, ok := rdx.GetAllValues(data.PrefixEnvProperty, prefixName); ok {
			for _, e := range env {
				summary[name] = append(summary[name], e)
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
