package cli

import (
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"strings"
)

func UpdateHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	all := q.Has("all")
	verbose := q.Has("verbose")
	force := q.Has("force")

	return Update(id, all, verbose, force)
}

func Update(id string, all, verbose, force bool) error {

	var updateMsg string
	switch all {
	case false:
		updateMsg = fmt.Sprintf("updating %s...", id)
	case true:
		updateMsg = fmt.Sprintf("updating all products...")
	}

	ua := nod.NewProgress(updateMsg)
	defer ua.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	updatedIdsInstallInfo, err := checkProductsUpdates(id, rdx, all, force)
	if err != nil {
		return err
	}

	for updatedId, installedInfoSlice := range updatedIdsInstallInfo {
		for _, installedInfo := range installedInfoSlice {

			installedInfo.verbose = verbose
			installedInfo.force = true

			if err = Install(updatedId, installedInfo); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkProductsUpdates(id string, rdx redux.Writeable, all, force bool) (map[string][]*InstallInfo, error) {

	cpua := nod.NewProgress("checking for products updates...")
	defer cpua.Done()

	if err := rdx.MustHave(data.InstallInfoProperty); err != nil {
		return nil, err
	}

	checkIds := make([]string, 0)
	if id != "" {
		checkIds = append(checkIds, id)
	}

	if all {
		for installedId := range rdx.Keys(data.InstallInfoProperty) {
			checkIds = append(checkIds, installedId)
		}
	}

	cpua.TotalInt(len(checkIds))

	updatedIdInstalledInfo := make(map[string][]*InstallInfo)

	for _, checkId := range checkIds {
		if uii, err := checkProductUpdates(checkId, rdx, force); err == nil && len(uii) > 0 {
			updatedIdInstalledInfo[id] = uii
		} else if err != nil {
			return nil, err
		}

		cpua.Increment()
	}

	updatedIds := make([]string, 0, len(updatedIdInstalledInfo))
	for uid := range updatedIdInstalledInfo {
		updatedIds = append(updatedIds, uid)
	}

	if len(updatedIdInstalledInfo) > 0 {
		cpua.EndWithResult("found updates for: %s", strings.Join(updatedIds, ","))
	} else {
		cpua.EndWithResult("all products are up to date")
	}

	return updatedIdInstalledInfo, nil

}

func checkProductUpdates(id string, rdx redux.Writeable, force bool) ([]*InstallInfo, error) {

	cpua := nod.Begin(" checking product updates for %s...", id)
	defer cpua.Done()

	updatedInstalledInfo := make([]*InstallInfo, 0)

	if installedInfoLines, ok := rdx.GetAllValues(data.InstallInfoProperty, id); ok {

		for _, line := range installedInfoLines {

			installedInfo, err := parseInstallInfo(line)
			if err != nil {
				return nil, err
			}

			if updated, err := isInstalledInfoUpdated(id, installedInfo, rdx, force); updated && err == nil {
				updatedInstalledInfo = append(updatedInstalledInfo, installedInfo)
			} else if err != nil {
				return nil, err
			}

		}

	}

	return updatedInstalledInfo, nil

}

func isInstalledInfoUpdated(id string, installedInfo *InstallInfo, rdx redux.Writeable, force bool) (bool, error) {

	iiiua := nod.Begin(" checking %s %s-%s version...", id, installedInfo.OperatingSystem, installedInfo.LangCode)
	defer iiiua.Done()

	latestProductDetails, err := getProductDetails(id, rdx, true)
	if err != nil {
		return false, err
	}

	installedVersion := installedInfo.Version
	latestVersion := productDetailsVersion(latestProductDetails, installedInfo)

	if installedVersion == "" && !force {
		iiiua.EndWithResult("cannot determine installed version")
		return false, nil
	}

	if latestVersion == "" && !force {
		iiiua.EndWithResult("cannot determine latest version")
		return false, nil
	}

	if installedVersion == latestVersion {
		iiiua.EndWithResult("already at the latest version: %s", installedVersion)
		return false, nil
	} else {
		iiiua.EndWithResult("found update to install: %s -> %s", installedVersion, latestVersion)
		return true, nil
	}
}

func productDetailsVersion(productDetails *vangogh_integration.ProductDetails, ii *InstallInfo) string {
	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterDownloadTypes(vangogh_integration.Installer).
		FilterLanguageCodes(ii.LangCode)

	var version string
	for ii, dl := range dls {
		if ii == 0 {
			version = dl.Version
		}
	}

	return version
}
