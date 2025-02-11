package cli

import (
	"encoding/json"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path/filepath"
)

func UpdateHandler(u *url.URL) error {

	q := u.Query()
	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	all := q.Has("all")

	return Update(data.CurrentOs(), langCode, all, ids...)
}

func Update(operatingSystem vangogh_integration.OperatingSystem, langCode string, all bool, ids ...string) error {

	ua := nod.NewProgress("updating installed products on %s...", operatingSystem.String())
	defer ua.EndWithResult("done")

	installedMetadataDir, err := pathways.GetAbsRelDir(data.InstalledMetadata)
	if err != nil {
		return ua.EndWithError(err)
	}

	osLangInstalledMetadataDir := filepath.Join(installedMetadataDir, data.OsLangCode(operatingSystem, langCode))

	kvOsLangInstalledMetadata, err := kevlar.New(osLangInstalledMetadataDir, kevlar.JsonExt)
	if err != nil {
		return ua.EndWithError(err)
	}

	if all {
		for id := range kvOsLangInstalledMetadata.Keys() {
			ids = append(ids, id)
		}
	}

	ua.TotalInt(len(ids))

	for _, id := range ids {
		if err = checkProductUpdates(id, operatingSystem, langCode, kvOsLangInstalledMetadata); err != nil {
			return ua.EndWithError(err)
		}
		ua.Increment()
	}

	return nil
}

func checkProductUpdates(id string,
	operatingSystem vangogh_integration.OperatingSystem,
	langCode string,
	kvOsLangInstalledMetadata kevlar.KeyValues) error {

	cpua := nod.Begin(" checking product updates for %s...", id)
	defer cpua.EndWithResult("done")

	if !kvOsLangInstalledMetadata.Has(id) {
		cpua.EndWithResult("not installed on %s", operatingSystem)
		return nil
	}

	rcInstalledMetadata, err := kvOsLangInstalledMetadata.Get(id)
	if err != nil {
		return cpua.EndWithError(err)
	}
	defer rcInstalledMetadata.Close()

	var installedMetadata vangogh_integration.TheoMetadata
	if err = json.NewDecoder(rcInstalledMetadata).Decode(&installedMetadata); err != nil {
		return cpua.EndWithError(err)
	}

	latestMetadata, err := getTheoMetadata(id, true)
	if err != nil {
		return cpua.EndWithError(err)
	}

	installedVersion := metadataVersion(&installedMetadata, operatingSystem, langCode)
	latestVersion := metadataVersion(latestMetadata, operatingSystem, langCode)

	if installedVersion == latestVersion {
		cpua.EndWithResult("already at the latest version: %s", installedVersion)
		return nil
	}

	reduxDir, err := pathways.GetAbsRelDir(vangogh_integration.Redux)
	if err != nil {
		return cpua.EndWithError(err)
	}

	rdx, err := redux.NewReader(reduxDir, data.InstallParametersProperty)
	if err != nil {
		return cpua.EndWithError(err)
	}

	var ip *installParameters

	if allInstallParameters, ok := rdx.GetAllValues(data.InstallParametersProperty, id); ok {
		ip = filterInstallParameters(operatingSystem, langCode, allInstallParameters...)
	}

	if ip == nil {
		ip = defaultInstallParameters(operatingSystem)
	}

	cpua.EndWithResult("found update to install: %s -> %s", installedVersion, latestVersion)

	return Install(ip, true, id)
}
