package cli

import (
	"crypto/md5"
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
)

func ValidateHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, _ := OsLangCodeDownloadType(u)

	return Validate(ids, operatingSystems, langCodes)
}

func Validate(ids []string, operatingSystems []vangogh_local_data.OperatingSystem, langCodes []string) error {

	PrintParams(ids, operatingSystems, langCodes, nil)

	va := nod.NewProgress("validating downloads...")
	defer va.End()

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
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

		if _, links, err := GetTitleDownloadLinks(id, operatingSystems, langCodes, nil, kvdm, false); err == nil {
			for _, dl := range links {
				if err = validateLink(id, dl, downloadsDir); err != nil {
					return va.EndWithError(err)
				}
			}
		} else {
			return va.EndWithError(err)
		}
	}

	va.EndWithResult("done")

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
