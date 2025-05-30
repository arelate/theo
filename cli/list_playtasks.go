package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"strings"
)

func ListPlaytasksHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	return ListPlaytasks(id, langCode)
}

func ListPlaytasks(id string, langCode string) error {

	lpta := nod.Begin("listing playtasks for %s...", id)
	defer lpta.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	absGogGameInfoPath, err := findGogGameInfoPath(id, langCode, rdx)
	if err != nil {
		return err
	}

	gogGameInfo, err := getGogGameInfo(absGogGameInfoPath)
	if err != nil {
		return err
	}

	playtasksSummary := make(map[string][]string)

	for _, pt := range gogGameInfo.PlayTasks {
		list := make([]string, 0)
		if pt.Arguments != "" {
			list = append(list, "arguments:"+pt.Arguments)
		}
		list = append(list, "category:"+pt.Category)
		if pt.IsPrimary {
			list = append(list, "isPrimary:true")
		}
		if len(pt.Languages) > 0 {
			list = append(list, "languages:"+strings.Join(pt.Languages, ","))
		}
		if pt.Path != "" {
			list = append(list, "path:"+pt.Path)
		}
		list = append(list, "type:"+pt.Type)
		if pt.WorkingDir != "" {
			list = append(list, "workingDir:"+pt.WorkingDir)
		}
		if pt.Link != "" {
			list = append(list, "link:"+pt.Link)
		}

		playtasksSummary["name:"+pt.Name] = list
	}

	lpta.EndWithSummary("found the following playtasks:", playtasksSummary)

	return nil
}
