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
	defer lia.EndWithResult("done")

	vangogh_integration.PrintParams(nil,
		[]vangogh_integration.OperatingSystem{os},
		[]string{langCode},
		nil,
		false)

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return lia.EndWithError(err)
	}

	osLangInstalledMetadataDir := filepath.Join(installedMetadataDir, os.String(), langCode)

	kvOsLangInstalledMetadata, err := kevlar.New(osLangInstalledMetadataDir, kevlar.JsonExt)
	if err != nil {
		return lia.EndWithError(err)
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

	for id := range kvOsLangInstalledMetadata.Keys() {

		installedMetadata, err := getInstalledMetadata(id, kvOsLangInstalledMetadata)
		if err != nil {
			return lia.EndWithError(err)
		}

		name := fmt.Sprintf("%s (%s)", installedMetadata.Title, id)

		dls := installedMetadata.DownloadLinks.
			FilterOperatingSystems(os).
			FilterDownloadTypes(vangogh_integration.Installer).
			FilterLanguageCodes(langCode)

		var version string
		var size int
		for ii, dl := range dls {
			if ii == 0 {
				version = dl.Version
			}
			size += dl.EstimatedBytes
		}

		summary[name] = append(summary[name], "size: "+fmtBytes(size), "ver.: "+version)

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

func getInstalledMetadata(id string, kvInstalled kevlar.KeyValues) (*vangogh_integration.TheoMetadata, error) {
	rcInstalled, err := kvInstalled.Get(id)
	if err != nil {
		return nil, err
	}
	defer rcInstalled.Close()

	var installedMetadata vangogh_integration.TheoMetadata

	if err = json.NewDecoder(rcInstalled).Decode(&installedMetadata); err != nil {
		return nil, err
	}

	return &installedMetadata, nil
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
