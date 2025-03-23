package cli

import (
	"encoding/json"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path/filepath"
)

func ListInstalledHandler(u *url.URL) error {
	langCode := defaultLangCode
	if u.Query().Has(vangogh_integration.LanguageCodeProperty) {
		langCode = u.Query().Get(vangogh_integration.LanguageCodeProperty)
	}
	return ListInstalled(data.CurrentOs(), langCode)
}

func ListInstalled(os vangogh_integration.OperatingSystem, langCode string) error {

	lia := nod.Begin("listing installed %s products...", os)
	defer lia.Done()

	vangogh_integration.PrintParams(nil,
		[]vangogh_integration.OperatingSystem{os},
		[]string{langCode},
		nil,
		false)

	installedManifestsDir, err := pathways.GetAbsRelDir(data.InstalledManifests)
	if err != nil {
		return err
	}

	osLangInstalledManifestsDir := filepath.Join(installedManifestsDir, data.OsLangCode(os, langCode))

	kvOsLangInstalledManifests, err := kevlar.New(osLangInstalledManifestsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	reduxDir, err := pathways.GetAbsRelDir(vangogh_integration.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir, data.InstallParametersProperty)
	if err != nil {
		return err
	}

	summary := make(map[string][]string)

	for id := range kvOsLangInstalledManifests.Keys() {

		installedManifest, err := getInstalledManifest(id, kvOsLangInstalledManifests)
		if err != nil {
			return err
		}

		name := fmt.Sprintf("%s (%s)", installedManifest.Title, id)
		version := manifestVersion(installedManifest, os, langCode)
		estimatedBytes := manifestEstimatedBytes(installedManifest, os, langCode)

		summary[name] = append(summary[name], "size: "+fmtBytes(estimatedBytes), "ver.: "+version)

		if installParams, ok := rdx.GetAllValues(data.InstallParametersProperty, id); ok {
			for _, ips := range installParams {
				summary[name] = append(summary[name], "par.: "+ips)
			}
		} else {
			summary[name] = append(summary[name], "par.: (default) "+defaultInstallParameters(os).String())
		}
	}

	if len(summary) == 0 {
		lia.EndWithResult("found nothing")
	} else {
		lia.EndWithSummary("found the following products:", summary)
	}

	return nil
}

func getInstalledManifest(id string, kvInstalledManifests kevlar.KeyValues) (*vangogh_integration.DownloadsManifest, error) {
	rcInstalled, err := kvInstalledManifests.Get(id)
	if err != nil {
		return nil, err
	}
	defer rcInstalled.Close()

	var installedManifest vangogh_integration.DownloadsManifest

	if err = json.NewDecoder(rcInstalled).Decode(&installedManifest); err != nil {
		return nil, err
	}

	return &installedManifest, nil
}

func fmtBytes(b int) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func manifestVersion(downloadsManifest *vangogh_integration.DownloadsManifest, operatingSystem vangogh_integration.OperatingSystem, langCode string) string {
	dls := downloadsManifest.DownloadLinks.
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

func manifestEstimatedBytes(downloadsManifest *vangogh_integration.DownloadsManifest, operatingSystem vangogh_integration.OperatingSystem, langCode string) int {
	dls := downloadsManifest.DownloadLinks.
		FilterOperatingSystems(operatingSystem).
		FilterDownloadTypes(vangogh_integration.Installer).
		FilterLanguageCodes(langCode)

	var size int
	for _, dl := range dls {
		size += dl.EstimatedBytes
	}

	return size
}
