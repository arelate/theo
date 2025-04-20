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

func pinInstalledDetails(operatingSystems []vangogh_integration.OperatingSystem, langCode string, force bool, ids ...string) error {

	pida := nod.NewProgress("pinning product details as installed...")
	defer pida.Done()

	vangogh_integration.PrintParams(ids, operatingSystems, []string{langCode}, nil, false)

	productDetailsDir, err := pathways.GetAbsRelDir(data.ProductDetails)
	if err != nil {
		return err
	}

	kvProductDetails, err := kevlar.New(productDetailsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	pida.TotalInt(len(ids))

	for _, id := range ids {

		if err = pinProductInstalledDetails(id, operatingSystems, langCode, kvProductDetails, force); err != nil {
			return err
		}

		pida.Increment()

	}

	return nil
}

func pinProductInstalledDetails(id string,
	operatingSystems []vangogh_integration.OperatingSystem,
	langCode string,
	kvProductDetails kevlar.KeyValues,
	force bool) error {

	for _, os := range operatingSystems {
		if err := osPinInstalledDetails(id, os, langCode, kvProductDetails, force); err != nil {
			return err
		}
	}

	return nil
}

func osPinInstalledDetails(id string,
	operatingSystem vangogh_integration.OperatingSystem,
	langCode string,
	kvProductDetails kevlar.KeyValues,
	force bool) error {

	pimoa := nod.Begin(" pinning product details as installed...")
	defer pimoa.Done()

	installedDetailsDir, err := pathways.GetAbsRelDir(data.InstalledDetails)
	if err != nil {
		return err
	}

	osLangInstalledDetailsDir := filepath.Join(installedDetailsDir, data.OsLangCode(operatingSystem, langCode))

	kvOsLangInstalledDetails, err := kevlar.New(osLangInstalledDetailsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	if !kvProductDetails.Has(id) {
		return errors.New("product details not found for: " + id)
	}

	if kvOsLangInstalledDetails.Has(id) && !force {
		return nil
	}

	src, err := kvProductDetails.Get(id)
	if err != nil {
		return err
	}

	defer src.Close()

	return kvOsLangInstalledDetails.Set(id, src)
}
