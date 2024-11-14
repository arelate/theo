package cli

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

func ValidateHandler(u *url.URL) error {

	q := u.Query()

	var ids []string
	if q.Has(vangogh_local_data.IdProperty) {
		ids = strings.Split(q.Get(vangogh_local_data.IdProperty), ",")
	}

	operatingSystems := vangogh_local_data.OperatingSystemsFromUrl(u)
	if len(operatingSystems) == 0 {
		switch runtime.GOOS {
		case "windows":
			operatingSystems = append(operatingSystems, vangogh_local_data.Windows)
		case "darwin":
			operatingSystems = append(operatingSystems, vangogh_local_data.MacOS)
		case "linux":
			operatingSystems = append(operatingSystems, vangogh_local_data.Windows)
		}
	}

	var langCodes []string
	if q.Has(vangogh_local_data.LanguageCodeProperty) {
		langCodes = strings.Split(q.Get(vangogh_local_data.LanguageCodeProperty), ",")
	}
	if len(langCodes) == 0 {
		langCodes = append(langCodes, defaultLangCode)
	}

	return Validate(ids, operatingSystems, langCodes)
}

func Validate(ids []string, operatingSystems []vangogh_local_data.OperatingSystem, langCodes []string) error {
	va := nod.NewProgress("validating downloads...")
	defer va.End()

	ddp, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return va.EndWithError(err)
	}

	dmdp, err := pathways.GetAbsRelDir(data.DownloadsMetadata)
	if err != nil {
		return va.EndWithError(err)
	}

	kvdm, err := kevlar.NewKeyValues(dmdp, kevlar.JsonExt)
	if err != nil {
		return va.EndWithError(err)
	}

	for _, id := range ids {
		if err = validateDownload(id, operatingSystems, langCodes, ddp, kvdm); err != nil {
			return va.EndWithError(err)
		}
	}

	va.EndWithResult("done")

	return nil
}

func validateDownload(id string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadsDir string,
	kv kevlar.KeyValues) error {

	dmrc, err := kv.Get(id)
	if err != nil {
		return err
	}
	defer dmrc.Close()

	var downloadMetadata vangogh_local_data.DownloadMetadata
	if err := json.NewDecoder(dmrc).Decode(&downloadMetadata); err != nil {
		return err
	}

	for _, dl := range downloadMetadata.DownloadLinks {

		os := vangogh_local_data.ParseOperatingSystem(dl.OS)
		if !slices.Contains(operatingSystems, os) {
			continue
		}
		if !slices.Contains(langCodes, dl.LanguageCode) {
			continue
		}

		if err = validateLink(id, dl, downloadsDir); err != nil {
			return err
		}
	}

	return nil
}

func validateLink(id string, dl vangogh_local_data.DownloadLink, downloadsDir string) error {

	dla := nod.NewProgress(" - %s...", dl.LocalFilename)
	defer dla.End()

	absDownloadPath := filepath.Join(downloadsDir, id, dl.LocalFilename)

	var stat os.FileInfo
	var err error

	if stat, err = os.Stat(absDownloadPath); os.IsNotExist(err) {
		dla.EndWithResult("not present")
		return nil
	}

	if dl.Md5 == "" {
		dla.EndWithResult("missing md5")
		return nil
	}

	dla.Total(uint64(stat.Size()))

	localFile, err := os.Open(absDownloadPath)
	if err != nil {
		return dla.EndWithError(err)
	}

	h := md5.New()
	if err = dolo.CopyWithProgress(h, localFile, dla); err != nil {
		return dla.EndWithError(err)
	}

	computedMd5 := fmt.Sprintf("%x", h.Sum(nil))
	if dl.Md5 == computedMd5 {
		dla.EndWithResult("valid md5")
	} else {
		dla.EndWithResult("md5 mismatch")
	}

	return nil
}
