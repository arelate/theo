package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

func ListInstalledHandler(u *url.URL) error {

	q := u.Query()

	var operatingSystems []vangogh_integration.OperatingSystem
	if q.Has(vangogh_integration.OperatingSystemsProperty) {
		operatingSystems = vangogh_integration.ParseManyOperatingSystems(strings.Split(q.Get(vangogh_integration.OperatingSystemsProperty), ","))
	}

	if len(operatingSystems) == 0 {
		for _, os := range vangogh_integration.AllOperatingSystems() {
			if os == vangogh_integration.AnyOperatingSystem {
				continue
			}
			operatingSystems = append(operatingSystems, os)
		}
	}

	var langCodes []string
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCodes = append(langCodes, strings.Split(q.Get(vangogh_integration.LanguageCodeProperty), ",")...)
	}

	if len(langCodes) == 0 {
		langCodes = append(langCodes, defaultLangCode)
	}

	return ListInstalled(operatingSystems, langCodes)
}

func ListInstalled(operatingSystems []vangogh_integration.OperatingSystem, langCodes []string) error {

	lia := nod.Begin("listing installed products...")
	defer lia.Done()

	vangogh_integration.PrintParams(nil, operatingSystems, langCodes, nil, false)

	installedDetailsDir, err := pathways.GetAbsRelDir(data.InstalledDetails)
	if err != nil {
		return err
	}

	reduxDir, err := pathways.GetAbsRelDir(vangogh_integration.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir,
		data.InstallParametersProperty,
		data.InstallDateProperty)
	if err != nil {
		return err
	}

	summary := make(map[string][]string)

	for _, os := range operatingSystems {
		for _, langCode := range langCodes {

			osLangInstalledDetailsDir := filepath.Join(installedDetailsDir, data.OsLangCode(os, langCode))

			var kvOsLangInstalledDetails kevlar.KeyValues
			kvOsLangInstalledDetails, err = kevlar.New(osLangInstalledDetailsDir, kevlar.JsonExt)
			if err != nil {
				return err
			}

			for id := range kvOsLangInstalledDetails.Keys() {

				installedDetails, err := getInstalledDetails(id, kvOsLangInstalledDetails)
				if err != nil {
					return err
				}

				name := fmt.Sprintf("%s (%s)", installedDetails.Title, id)
				version := productDetailsVersion(installedDetails, os, langCode)
				estimatedBytes := productDetailsEstimatedBytes(installedDetails, os, langCode)

				summary[name] = append(summary[name],
					"size: "+vangogh_integration.FormatBytes(estimatedBytes),
					"ver.: "+version)

				if installParams, ok := rdx.GetAllValues(data.InstallParametersProperty, id); ok {
					for _, ips := range installParams {
						summary[name] = append(summary[name], "par.: "+ips)
					}
				} else {
					summary[name] = append(summary[name], "par.: (default) "+defaultInstallParameters(os).String())
				}

				if ids, ok := rdx.GetLastVal(data.InstallDateProperty, id); ok && ids != "" {
					if installDate, err := time.Parse(time.RFC3339, ids); err == nil {
						summary[name] = append(summary[name], "installed: "+installDate.Local().Format(time.DateTime))
					}
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

func getInstalledDetails(id string, kvInstalledDetails kevlar.KeyValues) (*vangogh_integration.ProductDetails, error) {

	rcInstalledDetails, err := kvInstalledDetails.Get(id)
	if err != nil {
		return nil, err
	}
	defer rcInstalledDetails.Close()

	var installedDetails vangogh_integration.ProductDetails

	if err = json.NewDecoder(rcInstalledDetails).Decode(&installedDetails); err != nil {
		return nil, err
	}

	return &installedDetails, nil
}

func productDetailsVersion(productDetails *vangogh_integration.ProductDetails, operatingSystem vangogh_integration.OperatingSystem, langCode string) string {
	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(operatingSystem).
		FilterDownloadTypes(vangogh_integration.Installer).
		FilterLanguageCodes(langCode)

	var version string
	for ii, dl := range dls {
		if ii == 0 {
			version = dl.Version
		}
	}

	return version
}

func productDetailsEstimatedBytes(productDetails *vangogh_integration.ProductDetails, operatingSystem vangogh_integration.OperatingSystem, langCode string) int64 {
	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(operatingSystem).
		FilterDownloadTypes(vangogh_integration.Installer).
		FilterLanguageCodes(langCode)

	var size int64
	for _, dl := range dls {
		size += int64(dl.EstimatedBytes)
	}

	return size
}
