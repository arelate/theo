package cli

import (
	"bytes"
	"crypto/md5"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/arelate/southern_light/steam_grid"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/author"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func vangoghGetProductDetails(id string, rdx redux.Writeable, force bool) (*vangogh_integration.ProductDetails, error) {

	gpda := nod.NewProgress(" getting vangogh product details for %s...", id)
	defer gpda.Done()

	productDetailsDir := data.Pwd.AbsRelDirPath(data.ProductDetails, vangogh_integration.Metadata)

	kvProductDetails, err := kevlar.New(productDetailsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	var pd *vangogh_integration.ProductDetails
	if pd, err = vangoghReadLocalProductDetails(id, kvProductDetails); err != nil {
		return nil, err
	} else if pd != nil && !force {
		gpda.EndWithResult("read local")
		return pd, nil
	}

	if err = vangoghValidateSessionToken(rdx); err != nil {
		return nil, err
	}

	productDetails, err := vangoghFetchRemoteProductDetails(id, rdx, kvProductDetails)
	if err != nil {
		return nil, err
	}

	gpda.EndWithResult("fetched remote")

	if err = vangoghReduceProductDetails(id, productDetails, rdx); err != nil {
		return nil, err
	}

	return productDetails, nil
}

func vangoghReadLocalProductDetails(id string, kvProductDetails kevlar.KeyValues) (*vangogh_integration.ProductDetails, error) {

	if has := kvProductDetails.Has(id); !has {
		return nil, nil
	}

	tmReadCloser, err := kvProductDetails.Get(id)
	if err != nil {
		return nil, err
	}
	defer tmReadCloser.Close()

	var productDetails vangogh_integration.ProductDetails
	if err = json.UnmarshalRead(tmReadCloser, &productDetails); err != nil {
		return nil, err
	}

	return &productDetails, nil
}

func vangoghFetchRemoteProductDetails(id string, rdx redux.Readable, kvProductDetails kevlar.KeyValues) (*vangogh_integration.ProductDetails, error) {

	fra := nod.Begin(" fetching vangogh product details from the origin for %s...", id)
	defer fra.Done()

	query := url.Values{
		vangogh_integration.IdProperty: {id},
	}

	req, err := data.VangoghApiRequest(http.MethodGet, data.ApiProductDetailsPath, query, rdx)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New("error fetching product details: " + resp.Status)
	}

	var bts []byte
	buf := bytes.NewBuffer(bts)
	tr := io.TeeReader(resp.Body, buf)

	if err = kvProductDetails.Set(id, tr); err != nil {
		return nil, err
	}

	var productDetails vangogh_integration.ProductDetails
	if err = json.UnmarshalRead(buf, &productDetails); err != nil {
		return nil, err
	}

	return &productDetails, nil
}

func vangoghReduceProductDetails(id string, productDetails *vangogh_integration.ProductDetails, rdx redux.Writeable) error {

	rpda := nod.Begin(" reducing vangogh product details...")
	defer rpda.Done()

	propertyValues := make(map[string][]string)

	oss := make([]string, 0, len(productDetails.OperatingSystems))
	for _, os := range productDetails.OperatingSystems {
		oss = append(oss, os.String())
	}

	reductionProperties := []string{
		vangogh_integration.GogSteamAppIdProperty,
		vangogh_integration.GogTitleProperty,
		vangogh_integration.OperatingSystemsProperty,
		vangogh_integration.GogDevelopersProperty,
		vangogh_integration.GogPublishersProperty,
		vangogh_integration.GogVerticalImageProperty,
		vangogh_integration.GogImageProperty,
		vangogh_integration.GogHeroProperty,
		vangogh_integration.GogLogoProperty,
		vangogh_integration.GogIconProperty,
		vangogh_integration.GogIconSquareProperty,
		vangogh_integration.GogBackgroundProperty,
	}

	for _, property := range reductionProperties {

		var values []string

		switch property {
		case vangogh_integration.GogSteamAppIdProperty:
			values = []string{productDetails.SteamAppId}
		case vangogh_integration.GogTitleProperty:
			values = []string{productDetails.Title}
		case vangogh_integration.OperatingSystemsProperty:
			values = oss
		case vangogh_integration.GogDevelopersProperty:
			values = productDetails.Developers
		case vangogh_integration.GogPublishersProperty:
			values = productDetails.Publishers
		case vangogh_integration.GogVerticalImageProperty:
			values = []string{productDetails.Images.VerticalImage}
		case vangogh_integration.GogImageProperty:
			values = []string{productDetails.Images.Image}
		case vangogh_integration.GogHeroProperty:
			values = []string{productDetails.Images.Hero}
		case vangogh_integration.GogLogoProperty:
			values = []string{productDetails.Images.Logo}
		case vangogh_integration.GogIconProperty:
			values = []string{productDetails.Images.Icon}
		case vangogh_integration.GogIconSquareProperty:
			values = []string{productDetails.Images.IconSquare}
		case vangogh_integration.GogBackgroundProperty:
			values = []string{productDetails.Images.Background}
		}

		if len(values) == 1 && values[0] == "" {
			values = nil
		}

		if len(values) > 0 {
			propertyValues[property] = values
		}
	}

	for property, values := range propertyValues {
		if err := rdx.ReplaceValues(property, id, values...); err != nil {
			return err
		}
	}

	return nil
}

func vangoghGetAvailableProducts(force bool) ([]vangogh_integration.AvailableProduct, error) {

	vlapa := nod.Begin("getting available vangogh products...")
	defer vlapa.Done()

	availableProductsDir := data.Pwd.AbsRelDirPath(data.AvailableProducts, data.Metadata)
	kvAvailableProducts, err := kevlar.New(availableProductsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	vangoghApKey := originAvailableProductsKey(data.VangoghOrigin, vangogh_integration.AnyOperatingSystem)

	if !kvAvailableProducts.Has(vangoghApKey) || force {
		if err = vangoghFetchAvailableProducts(kvAvailableProducts); err != nil {
			return nil, err
		}
	}

	rcAvailableProducts, err := kvAvailableProducts.Get(vangoghApKey)
	if err != nil {
		return nil, err
	}
	defer rcAvailableProducts.Close()

	var availableProducts []vangogh_integration.AvailableProduct
	if err = json.UnmarshalRead(rcAvailableProducts, &availableProducts); err != nil {
		return nil, err
	}

	return availableProducts, nil
}

func vangoghFetchAvailableProducts(kvAvailableProducts kevlar.KeyValues) error {

	vgapa := nod.Begin(" fetching vangogh available products...")
	defer vgapa.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.VangoghProperties()...)
	if err != nil {
		return err
	}

	if err = vangoghValidateSessionToken(rdx); err != nil {
		return err
	}

	req, err := data.VangoghApiRequest(http.MethodGet, data.ApiAvailableProducts, nil, rdx)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New("error fetching available products: " + resp.Status)
	}

	vangoghApKey := originAvailableProductsKey(data.VangoghOrigin, vangogh_integration.AnyOperatingSystem)

	return kvAvailableProducts.Set(vangoghApKey, resp.Body)
}

func vangoghProductDetailsSize(productDetails *vangogh_integration.ProductDetails, ii *InstallInfo, manualUrlFilter ...string) int64 {
	var totalEstimatedBytes int64

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	for _, dl := range dls {
		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}
		totalEstimatedBytes += dl.EstimatedBytes
	}

	return totalEstimatedBytes
}

func vangoghUninstallProduct(id string, ii *InstallInfo, rdx redux.Writeable) error {

	oupa := nod.Begin(" uninstalling %s %s-%s...", id, ii.OperatingSystem, ii.LangCode)
	defer oupa.Done()

	if err := removeInventoriedFiles(id, ii, rdx); err != nil {
		return err
	}

	return nil
}

func vangoghShortcutAssets(productDetails *vangogh_integration.ProductDetails, rdx redux.Readable) (map[steam_grid.Asset]*url.URL, error) {

	shortcutAssets := make(map[steam_grid.Asset]*url.URL)

	for _, asset := range steam_grid.ShortcutAssets {

		var imageId string
		switch asset {
		case steam_grid.Header:
			imageId = productDetails.Images.Image
		case steam_grid.LibraryCapsule:
			imageId = productDetails.Images.VerticalImage
		case steam_grid.LibraryHero:
			if productDetails.Images.Hero != "" {
				imageId = productDetails.Images.Hero
			} else {
				imageId = productDetails.Images.Background
			}
		case steam_grid.LibraryLogo:
			imageId = productDetails.Images.Logo
		case steam_grid.ClientIcon:
			if productDetails.Images.IconSquare != "" {
				imageId = productDetails.Images.IconSquare
			} else {
				imageId = productDetails.Images.Icon
			}
		default:
			return nil, errors.New("unexpected shortcut asset " + asset.String())
		}

		if imageId != "" {
			imageQuery := url.Values{
				"id": {imageId},
			}

			vangoghImageUrl, err := data.VangoghUrl(data.ApiImagePath, imageQuery, rdx)
			if err != nil {
				return nil, err
			}

			shortcutAssets[asset] = vangoghImageUrl
		}
	}

	return shortcutAssets, nil

}

func vangoghSetupConnection(urlStr, username, password string, rdx redux.Writeable, reset bool) error {

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	if reset {
		if err := vangoghResetConnection(rdx); err != nil {
			return err
		}
	}

	if err := rdx.ReplaceValues(data.VangoghUrlProperty, data.VangoghUrlProperty, urlStr); err != nil {
		return err
	}

	if err := rdx.ReplaceValues(data.VangoghUsernameProperty, data.VangoghUsernameProperty, username); err != nil {
		return err
	}

	if err := vangoghUpdateSessionToken(password, rdx); err != nil {
		return err
	}

	return vangoghValidateSessionToken(rdx)
}

func vangoghResetConnection(rdx redux.Writeable) error {
	rvca := nod.Begin("resetting vangogh connection...")
	defer rvca.Done()

	for _, vp := range data.VangoghProperties() {
		if err := rdx.CutKeys(vp, vp); err != nil {
			return err
		}
	}

	return nil
}

func vangoghValidateSessionToken(rdx redux.Readable) error {

	tsa := nod.Begin("validating vangogh session token...")
	defer tsa.Done()

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	req, err := data.VangoghApiRequest(http.MethodPost, data.ApiAuthSessionPath, nil, rdx)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		msg := "session is not valid, please connect again"
		tsa.EndWithResult(msg)
		return errors.New(msg)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	var ste author.SessionTokenExpires

	if err = json.UnmarshalRead(resp.Body, &ste); err != nil {
		return err
	}

	utcNow := time.Now().UTC()

	if utcNow.Before(ste.Expires.Add(-1 * time.Hour * author.SessionNearExpirationHours)) {
		tsa.EndWithResult("session is valid")
		return nil
	} else {
		msg := "vangogh session expired or expires soon, connect to update"
		tsa.EndWithResult(msg)
		return errors.New(msg)
	}

}

func vangoghUpdateSessionToken(password string, rdx redux.Writeable) error {
	rsa := nod.Begin("updating vangogh session token...")
	defer rsa.Done()

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	var username string
	if up, ok := rdx.GetLastVal(data.VangoghUsernameProperty, data.VangoghUsernameProperty); ok && up != "" {
		username = up
	} else {
		return errors.New("username not found")
	}

	usernamePassword := url.Values{}
	usernamePassword.Set(author.UsernameParam, username)
	usernamePassword.Set(author.PasswordParam, password)

	req, err := data.VangoghApiRequest(http.MethodPost, data.ApiAuthUserPath, usernamePassword, rdx)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	var ste author.SessionTokenExpires

	if err = json.UnmarshalRead(resp.Body, &ste); err != nil {
		return err
	}

	if err = rdx.ReplaceValues(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty, ste.Token); err != nil {
		return err
	}

	if err = rdx.ReplaceValues(data.VangoghSessionExpiresProperty, data.VangoghSessionExpiresProperty, ste.Expires.Format(http.TimeFormat)); err != nil {
		return err
	}

	return nil
}

func vangoghUnpackPlace(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable) error {

	ipa := nod.Begin("installing %s %s-%s...", id, ii.OperatingSystem, ii.LangCode)
	defer ipa.Done()

	dls := originData.ProductDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		ipa.EndWithResult("no links are matching install params")
		return nil
	}

	dlcNames := make(map[string]any)

	for _, dl := range dls {
		if ii.OperatingSystem != dl.OperatingSystem ||
			ii.LangCode != dl.LanguageCode {
			continue
		}
		if dl.DownloadType == vangogh_integration.DLC {
			dlcNames[dl.Name] = nil
		}
	}

	if len(dlcNames) > 0 {
		ii.DownloadableContent = slices.Collect(maps.Keys(dlcNames))
	}

	// vangogh installation:
	// 1. check available space
	// 2. unpack installers (e.g. pkgutil on macOS, execute .sh on Linux; run setup on Windows)
	// 3. perform post-unpack actions (e.g. reduce bundleName on macOS)
	// 4. uninstall if installed directory exists and forcing install (will be used for updates)
	// 5. create inventory of unpacked files
	// 6. place (move unpacked to install folder)
	// 7. perform post-install actions (e.g. run post-install script and remove xattrs on macOS)
	// 8. cleanup unpack directory

	// 1
	installedAppsDir := data.Pwd.AbsDirPath(data.InstalledApps)

	if err := originHasFreeSpace(id, installedAppsDir, ii, originData); err != nil {
		return err
	}

	// 2
	unpackDir, err := vangoghGetUnpackDir(id, ii, rdx)
	if err != nil {
		return err
	}

	if err = vangoghUnpackInstallers(id, ii, dls, rdx, unpackDir); err != nil {
		return err
	}

	// 3
	if err = vangoghPostUnpackActions(id, ii, dls, unpackDir, rdx); err != nil {
		return err
	}

	// 4
	absInstalledDir, err := originOsInstalledPath(id, ii, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absInstalledDir); err == nil && ii.force {
		if err = vangoghUninstallProduct(id, ii, rdx); err != nil {
			return err
		}
	}

	// 5
	unpackedInventory, err := vangoghGetInventory(ii, dls, unpackDir)
	if err != nil {
		return err
	}

	if err = writeInventory(id, ii.LangCode, ii.OperatingSystem, rdx, unpackedInventory...); err != nil {
		return err
	}

	// 6
	if err = vangoghPlaceUnpackedFiles(id, ii, dls, rdx, unpackDir); err != nil {
		return err
	}

	// 7
	if err = vangoghPostInstallActions(id, ii, dls, rdx, unpackDir); err != nil {
		return err
	}

	// 8
	if err = os.RemoveAll(unpackDir); err != nil {
		return err
	}

	return nil
}

func vangoghGetUnpackDir(id string, ii *InstallInfo, rdx redux.Readable) (string, error) {

	unpackDir := filepath.Join(data.Pwd.AbsDirPath(data.Temp), id)

	switch ii.OperatingSystem {
	case vangogh_integration.Windows:
		switch data.CurrentOs() {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			absPrefixDir, err := data.AbsPrefixDir(id, ii.Origin, rdx)
			if err != nil {
				return "", err
			}
			return filepath.Join(absPrefixDir, prefixRelDriveCDir, "Temp", id), nil
		default:
			// do nothing
		}
	default:
		// do nothing
	}
	return unpackDir, nil
}

func vangoghUnpackInstallers(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Writeable, unpackDir string) error {

	if _, err := os.Stat(unpackDir); err == nil {
		if ii.force {
			if err = os.RemoveAll(unpackDir); err != nil {
				return err
			}
		} else {
			return nil
		}
	}

	if _, err := os.Stat(unpackDir); os.IsNotExist(err) {
		if err = os.MkdirAll(unpackDir, 0755); err != nil {
			return err
		}
	}

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsUnpackInstallers(id, dls, unpackDir, ii.force)
	case vangogh_integration.Linux:
		return linuxUnpackInstallers(id, dls, unpackDir)
	case vangogh_integration.Windows:
		switch data.CurrentOs() {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			return prefixUnpackInstallers(id, ii, dls, rdx, unpackDir)
		default:
			return ii.OperatingSystem.ErrUnsupported()
		}
	default:
		return ii.OperatingSystem.ErrUnsupported()
	}
}

func vangoghPostUnpackActions(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, unpackDir string, rdx redux.Writeable) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsReduceBundleNameProperty(id, dls, unpackDir, rdx)
	default:
		return nil
	}
}

func vangoghGetInventory(ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, unpackDir string) ([]string, error) {

	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsGetInventory(dls, unpackDir, ii.force)
	default:
		return getInventory(ii.OperatingSystem, dls, unpackDir)
	}
}

func vangoghPlaceUnpackedFiles(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Writeable, unpackDir string) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsPlaceUnpackedFiles(id, ii, dls, rdx, unpackDir, ii.force)
	case vangogh_integration.Linux:
		return linuxPlaceUnpackedFiles(id, ii, dls, rdx, unpackDir)
	case vangogh_integration.Windows:
		switch data.CurrentOs() {
		case vangogh_integration.MacOS:
			fallthrough
		case vangogh_integration.Linux:
			return prefixPlaceUnpackedFiles(id, ii, dls, rdx, unpackDir)
		default:
			return ii.OperatingSystem.ErrUnsupported()
		}
	default:
		return ii.OperatingSystem.ErrUnsupported()
	}
}

func vangoghPlaceUnpackedLinkPayload(link *vangogh_integration.ProductDownloadLink, absUnpackedPath, absInstallationPath string) error {

	mpda := nod.Begin(" placing unpacked %s files...", link.LocalFilename)
	defer mpda.Done()

	if _, err := os.Stat(absInstallationPath); os.IsNotExist(err) {
		if err = os.MkdirAll(absInstallationPath, 0755); err != nil {
			return err
		}
	}

	// enumerate all files in the payload directory
	relFiles, err := relWalkDir(absUnpackedPath)
	if err != nil {
		return err
	}

	for _, relFile := range relFiles {

		absSrcPath := filepath.Join(absUnpackedPath, relFile)

		absDstPath := filepath.Join(absInstallationPath, relFile)
		absDstDir, _ := filepath.Split(absDstPath)

		if _, err = os.Stat(absDstDir); os.IsNotExist(err) {
			if err = os.MkdirAll(absDstDir, 0755); err != nil {
				return err
			}
		}

		if err = os.Rename(absSrcPath, absDstPath); err != nil {
			return err
		}
	}

	return nil
}

func vangoghPostInstallActions(id string, ii *InstallInfo, dls vangogh_integration.ProductDownloadLinks, rdx redux.Readable, unpackDir string) error {
	switch ii.OperatingSystem {
	case vangogh_integration.MacOS:
		return macOsPostInstallActions(id, ii, dls, rdx, unpackDir, ii.force)
	default:
		return nil
	}
}

func vangoghDownloadData(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable, manualUrlFilter ...string) error {

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return err
	}

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	if err := originHasFreeSpace(id, downloadsDir, ii, originData, manualUrlFilter...); err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if token, ok := rdx.GetLastVal(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty); ok && token != "" {
		dc.SetAuthorizationBearer(token)
	}

	dls := originData.ProductDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		return errors.New("no links are matching operating params")
	}

	for _, dl := range dls {

		if dl.LocalFilename == "" {
			return errors.New("unresolved local filename for manual-url " + dl.ManualUrl)
		}

		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}

		if dl.ValidationStatus != vangogh_integration.ValidationStatusSuccess &&
			dl.ValidationStatus != vangogh_integration.ValidationStatusSelfValidated &&
			dl.ValidationStatus != vangogh_integration.ValidationStatusMissingChecksum {
			errMsg := fmt.Sprintf("%s validation status %s prevented download", dl.Name, dl.ValidationStatus)
			return errors.New(errMsg)
		}

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		query := url.Values{
			"manual-url":    {dl.ManualUrl},
			"id":            {id},
			"download-type": {dl.DownloadType.String()},
		}

		fileUrl, err := data.VangoghUrl(data.ApiFilePath, query, rdx)
		if err != nil {
			fa.EndWithResult(err.Error())
			continue
		}

		if err = dc.Download(fileUrl, ii.force, fa, downloadsDir, id, dl.LocalFilename); err != nil {
			fa.EndWithResult(err.Error())
			continue
		}

		fa.Done()
	}

	return nil
}

func vangoghRemoveProductDownloadLinks(id string,
	productDetails *vangogh_integration.ProductDetails,
	ii *InstallInfo,
	downloadsDir string) error {

	rdla := nod.Begin(" removing downloads for %s...", productDetails.Title)
	defer rdla.Done()

	idPath := filepath.Join(downloadsDir, id)
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		rdla.EndWithResult("product downloads dir not present")
		return nil
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		rdla.EndWithResult("no links are matching operating params")
		return nil
	}

	for _, dl := range dls {

		// if we don't do this - product downloads dir itself will be removed
		if dl.LocalFilename == "" {
			continue
		}

		path := filepath.Join(downloadsDir, id, dl.LocalFilename)

		fa := nod.NewProgress(" - %s...", dl.LocalFilename)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fa.EndWithResult("not present")
			continue
		}

		if err := os.Remove(path); err != nil {
			return err
		}

		fa.Done()
	}

	productDownloadsDir := filepath.Join(downloadsDir, id)
	if entries, err := os.ReadDir(productDownloadsDir); err == nil && len(entries) == 0 {
		rdda := nod.Begin(" removing empty product downloads directory...")
		if err = os.Remove(productDownloadsDir); err != nil {
			return err
		}
		rdda.Done()
	} else {
		return err
	}

	return nil
}

func vangoghGetExecTask(id string, ii *InstallInfo, rdx redux.Readable, et *execTask) (*execTask, error) {

	var err error
	if err = osConfirmRunnability(ii.OperatingSystem); err != nil {
		return nil, err
	}

	if ii.OperatingSystem == vangogh_integration.Windows && data.CurrentOs() != vangogh_integration.Windows {

		var absPrefixDir string
		if absPrefixDir, err = data.AbsPrefixDir(id, ii.Origin, rdx); err == nil {
			et.prefix = absPrefixDir
		} else {
			return nil, err
		}

		if et.exe != "" {
			return et, nil
		}
	}

	var absGogGameInfoPath string
	switch et.defaultLauncher {
	case false:
		absGogGameInfoPath, err = osFindGogGameInfo(id, ii, rdx)
		if err != nil {
			return nil, err
		}
	case true:
		// do nothing
	}

	switch absGogGameInfoPath {
	case "":
		var absDefaultLauncherPath string
		if absDefaultLauncherPath, err = osFindDefaultLauncher(id, ii, rdx); err != nil {
			return nil, err
		}
		if et, err = osExecTaskDefaultLauncher(absDefaultLauncherPath, ii.OperatingSystem, et); err != nil {
			return nil, err
		}
	default:
		if et, err = osExecTaskGogGameInfo(absGogGameInfoPath, ii.OperatingSystem, et); err != nil {
			return nil, err
		}
	}

	return et, nil
}

func vangoghValidateData(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable, manualUrlFilter ...string) error {
	va := nod.NewProgress("validating downloads...")
	defer va.Done()

	// always request new manual-url-checksums to avoid potentially reusing existing stale data
	manualUrlChecksums, err := getManualUrlChecksums(id, rdx, true)
	if err != nil {
		return err
	}

	// TODO: currently this never returns an error, consider replacing redownload loop with an error
	// and a parameter `no-validation`

	var mismatchedManualUrls []string
	if mismatchedManualUrls, err = vangoghValidateLinks(id, ii, manualUrlFilter, originData.ProductDetails, manualUrlChecksums); err != nil {
		return err
	} else if len(mismatchedManualUrls) > 0 {

		// redownload and revalidate any manual-urls that resulted in mismatched checksums

		ii.force = true

		if err = Download(id, ii, nil, mismatchedManualUrls...); err != nil {
			return err
		}

		if _, err = vangoghValidateLinks(id, ii, manualUrlFilter, originData.ProductDetails, manualUrlChecksums); err != nil {
			return err
		}
	}

	return nil
}

func vangoghValidateLinks(id string,
	ii *InstallInfo,
	manualUrlFilter []string,
	productDetails *vangogh_integration.ProductDetails,
	manualUrlChecksums map[string]string) ([]string, error) {

	vla := nod.NewProgress("validating %s...", productDetails.Title)
	defer vla.Done()

	downloadsDir := data.Pwd.AbsDirPath(data.Downloads)

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(ii.OperatingSystem).
		FilterLanguageCodes(ii.LangCode).
		FilterDownloadTypes(ii.DownloadTypes...)

	if len(dls) == 0 {
		return nil, errors.New("no links are matching operating params")
	}

	vla.TotalInt(len(dls))

	results := make([]ValidationResult, 0, len(dls))

	var mismatchedManualUrls []string

	for _, dl := range dls {
		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}

		vr, err := vangoghValidateLink(id, &dl, manualUrlChecksums[dl.ManualUrl], downloadsDir)
		if err != nil {
			vla.Error(err)
		}

		if vr == ValResMismatch {
			mismatchedManualUrls = append(mismatchedManualUrls, dl.ManualUrl)
		}

		results = append(results, vr)
	}

	vla.EndWithResult(summarizeValidationResults(results))

	return mismatchedManualUrls, nil
}

func vangoghValidateLink(id string, link *vangogh_integration.ProductDownloadLink, manualUrlMd5 string, downloadsDir string) (ValidationResult, error) {

	dla := nod.NewProgress(" - %s...", link.LocalFilename)
	defer dla.Done()

	absDownloadPath := filepath.Join(downloadsDir, id, link.LocalFilename)

	var stat os.FileInfo
	var err error

	if stat, err = os.Stat(absDownloadPath); os.IsNotExist(err) {
		dla.EndWithResult(ValResFileNotFound)
		return ValResFileNotFound, nil
	}

	if manualUrlMd5 == "" {
		dla.EndWithResult(ValResMissingChecksum)
		return ValResMissingChecksum, nil
	}

	dla.Total(uint64(stat.Size()))

	localFile, err := os.Open(absDownloadPath)
	if err != nil {
		return ValResError, err
	}

	h := md5.New()
	if err = dolo.CopyWithProgress(h, localFile, dla); err != nil {
		return ValResError, err
	}

	computedMd5 := fmt.Sprintf("%x", h.Sum(nil))
	if manualUrlMd5 == computedMd5 {
		dla.EndWithResult(ValResValid)
		return ValResValid, nil
	} else {
		dla.EndWithResult(ValResMismatch)
		return ValResMismatch, nil
	}
}
