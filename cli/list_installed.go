package cli

import (
	"errors"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"slices"
	"strings"
	"time"
)

func ListInstalledHandler(u *url.URL) error {

	q := u.Query()

	operatingSystem := vangogh_integration.AnyOperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystem = vangogh_integration.ParseOperatingSystem(q.Get(vangogh_integration.OperatingSystemsProperty))
	}

	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	ii := &InstallInfo{
		OperatingSystem: operatingSystem,
		LangCode:        langCode,
	}

	return ListInstalled(ii)
}

func ListInstalled(ii *InstallInfo) error {

	lia := nod.Begin("listing installed products for %s, %s...", ii.OperatingSystem, ii.LangCode)
	defer lia.Done()

	reduxDir, err := pathways.GetAbsRelDir(vangogh_integration.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir,
		vangogh_integration.TitleProperty,
		data.InstallInfoProperty,
		data.InstallDateProperty)
	if err != nil {
		return err
	}

	summary := make(map[string][]string)

	installedIds := slices.Collect(rdx.Keys(data.InstallInfoProperty))
	installedIds, err = rdx.Sort(installedIds, false, vangogh_integration.TitleProperty)
	if err != nil {
		return err
	}

	for _, id := range installedIds {

		title := id
		if tp, ok := rdx.GetLastVal(vangogh_integration.TitleProperty, id); ok {
			title = fmt.Sprintf("%s (%s)", tp, id)
		}

		var installedDate string
		if ids, ok := rdx.GetLastVal(data.InstallDateProperty, id); ok && ids != "" {
			if installDate, err := time.Parse(time.RFC3339, ids); err == nil {
				installedDate = installDate.Local().Format(time.DateTime)
			}
		}

		installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id)
		if !ok {
			return errors.New("install info not found for " + id)
		}

		for _, line := range installedInfoLines {

			installedInfo, err := parseInstallInfo(line)
			if err != nil {
				return err
			}

			infoLines := make([]string, 0)

			if (ii.OperatingSystem == vangogh_integration.AnyOperatingSystem ||
				installedInfo.OperatingSystem == ii.OperatingSystem) &&
				installedInfo.LangCode == ii.LangCode {

				infoLines = append(infoLines, "os: "+installedInfo.OperatingSystem.String())
				infoLines = append(infoLines, "lang: "+installedInfo.LangCode)
				infoLines = append(infoLines, "version: "+installedInfo.Version)
				if installedInfo.EstimatedBytes > 0 {
					infoLines = append(infoLines, "size: "+vangogh_integration.FormatBytes(installedInfo.EstimatedBytes))
				}

				summary[title] = append(summary[title], strings.Join(infoLines, "; "))
				if installedDate != "" {
					summary[title] = append(summary[title], "- installed: "+installedDate)
				}

			}

		}

	}

	if len(summary) == 0 {
		lia.EndWithResult("found nothing")
	} else {
		lia.EndWithSummary("found the following products:", summary)
	}

	return nil
}
