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
	"strings"
)

func UpdateHandler(u *url.URL) error {

	q := u.Query()
	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	all := q.Has("all")
	reveal := q.Has("reveal")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return Update(data.CurrentOs(), langCode, rdx, reveal, all, ids...)
}

func Update(operatingSystem vangogh_integration.OperatingSystem, langCode string, rdx redux.Writeable, reveal, all bool, ids ...string) error {

	ua := nod.NewProgress("updating installed %s products...", operatingSystem.String())
	defer ua.Done()

	updatedIds, err := filterUpdatedProducts(operatingSystem, langCode, rdx, all, ids...)
	if err != nil {
		return err
	}

	for _, id := range updatedIds {
		ip, err := loadInstallParameters(id, operatingSystem, langCode, rdx, reveal, true)
		if err != nil {
			return err
		}

		if err = Install(ip, id); err != nil {
			return err
		}
	}

	return nil
}

func filterUpdatedProducts(operatingSystem vangogh_integration.OperatingSystem, langCode string, rdx redux.Writeable, all bool, ids ...string) ([]string, error) {

	fupa := nod.NewProgress("filtering updated products...")
	defer fupa.Done()

	installedDetailsDir, err := pathways.GetAbsRelDir(data.InstalledDetails)
	if err != nil {
		return nil, err
	}

	osLangInstalledDetailsDir := filepath.Join(installedDetailsDir, data.OsLangCode(operatingSystem, langCode))

	kvOsLangInstalledDetails, err := kevlar.New(osLangInstalledDetailsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if all {
		for id := range kvOsLangInstalledDetails.Keys() {
			ids = append(ids, id)
		}
	}

	fupa.TotalInt(len(ids))

	updatedIds := make([]string, 0)

	for _, id := range ids {
		if updated, err := isProductUpdated(id, operatingSystem, langCode, rdx, kvOsLangInstalledDetails); err != nil {
			return nil, err
		} else if updated {
			updatedIds = append(updatedIds, id)
		}

		fupa.Increment()
	}

	if len(updatedIds) > 0 {
		fupa.EndWithResult("found updates for: %s", strings.Join(updatedIds, ","))
	} else {
		fupa.EndWithResult("no updates found")
	}

	return updatedIds, nil

}

func isProductUpdated(id string, operatingSystem vangogh_integration.OperatingSystem, langCode string, rdx redux.Writeable, kvOsLangInstalledDetails kevlar.KeyValues) (bool, error) {

	cpua := nod.Begin(" checking product updates for %s...", id)
	defer cpua.Done()

	if !kvOsLangInstalledDetails.Has(id) {
		cpua.EndWithResult("not installed on %s", operatingSystem)
		return false, nil
	}

	rcInstalledDetails, err := kvOsLangInstalledDetails.Get(id)
	if err != nil {
		return false, err
	}
	defer rcInstalledDetails.Close()

	var installedDetails vangogh_integration.ProductDetails
	if err = json.NewDecoder(rcInstalledDetails).Decode(&installedDetails); err != nil {
		return false, err
	}

	latestProductDetails, err := GetProductDetails(id, rdx, true)
	if err != nil {
		return false, err
	}

	installedVersion := productDetailsVersion(&installedDetails, operatingSystem, langCode)
	latestVersion := productDetailsVersion(latestProductDetails, operatingSystem, langCode)

	if installedVersion == latestVersion {
		cpua.EndWithResult("already at the latest version: %s", installedVersion)
		return false, nil
	} else {
		cpua.EndWithResult("found update to install: %s -> %s", installedVersion, latestVersion)
		return true, nil
	}
}
