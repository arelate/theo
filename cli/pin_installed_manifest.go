package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"path/filepath"
)

func pinInstalledManifests(operatingSystems []vangogh_integration.OperatingSystem, langCode string, force bool, ids ...string) error {

	pima := nod.NewProgress("pinning download manifests as installed...")
	defer pima.Done()

	vangogh_integration.PrintParams(ids, operatingSystems, []string{langCode}, nil, false)

	downloadsManifestsDir, err := pathways.GetAbsRelDir(data.DownloadsManifests)
	if err != nil {
		return err
	}

	kvDownloadsManifests, err := kevlar.New(downloadsManifestsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	pima.TotalInt(len(ids))

	for _, id := range ids {

		if err = pinProductInstalledManifest(id, operatingSystems, langCode, kvDownloadsManifests, force); err != nil {
			return err
		}

		pima.Increment()

	}

	return nil
}

func pinProductInstalledManifest(id string,
	operatingSystems []vangogh_integration.OperatingSystem,
	langCode string,
	kvDownloadsManifests kevlar.KeyValues,
	force bool) error {

	for _, os := range operatingSystems {
		if err := osPinInstalledManifest(id, os, langCode, kvDownloadsManifests, force); err != nil {
			return err
		}
	}

	return nil
}

func osPinInstalledManifest(id string,
	operatingSystem vangogh_integration.OperatingSystem,
	langCode string,
	kvDownloadsManifests kevlar.KeyValues,
	force bool) error {

	pimoa := nod.Begin(" pinning download manifest as installed...")
	defer pimoa.Done()

	installedManifestsDir, err := pathways.GetAbsRelDir(data.InstalledManifests)
	if err != nil {
		return err
	}

	osLangInstalledManifestsDir := filepath.Join(installedManifestsDir, data.OsLangCode(operatingSystem, langCode))

	kvOsLangInstalledManifests, err := kevlar.New(osLangInstalledManifestsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if hasDownloadManifest := kvDownloadsManifests.Has(id); !hasDownloadManifest {
		return errors.New("download manifest not found for: " + id)
	}

	if kvOsLangInstalledManifests.Has(id) && !force {
		return nil
	}

	src, err := kvDownloadsManifests.Get(id)
	if err != nil {
		return err
	}

	defer src.Close()

	return kvOsLangInstalledManifests.Set(id, src)
}
