package cli

import (
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
	"path/filepath"
)

const (
	pkgExt = ".pkg"
	exeExt = ".exe"
)

func ExtractHandler(u *url.URL) error {

	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)
	force := u.Query().Has("force")

	return Extract(ids, operatingSystems, langCodes, downloadTypes, force)
}

func Extract(ids []string,
	operatingSystems []vangogh_local_data.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_local_data.DownloadType,
	force bool) error {

	ea := nod.NewProgress("extracting installers game data...")
	defer ea.End()

	PrintParams(ids, operatingSystems, langCodes, downloadTypes)

	ea.TotalInt(len(ids))

	dmd, err := pathways.GetAbsRelDir(data.DownloadsMetadata)
	if err != nil {
		return ea.EndWithError(err)
	}

	kvdm, err := kevlar.NewKeyValues(dmd, kevlar.JsonExt)
	if err != nil {
		return ea.EndWithError(err)
	}

	for _, id := range ids {

		if title, links, err := GetTitleDownloadLinks(id, operatingSystems, langCodes, downloadTypes, kvdm, force); err == nil {
			if err = extractProductDownloadLinks(id, title, links, force); err != nil {
				return ea.EndWithError(err)
			}
		} else {
			return ea.EndWithError(err)
		}

		ea.Increment()
	}

	ea.EndWithResult("done")

	return nil
}

func extractProductDownloadLinks(id, title string, links []vangogh_local_data.DownloadLink, force bool) error {

	epdla := nod.NewProgress(" extracting %s...", title)
	defer epdla.End()

	extractsDir, err := pathways.GetAbsDir(data.Extracts)
	if err != nil {
		return epdla.EndWithError(err)
	}

	productExtractsDir := filepath.Join(extractsDir, id)

	for _, link := range links {

		linkOs := vangogh_local_data.ParseOperatingSystem(link.OS)
		linkExt := filepath.Ext(link.LocalFilename)

		if linkOs == vangogh_local_data.MacOS && linkExt == pkgExt {
			if err := extractPkgScripts(link, productExtractsDir); err != nil {
				return epdla.EndWithError(err)
			}
		}
		if linkOs == vangogh_local_data.Windows && linkExt == exeExt {
			if err := extractInnosetup(link, productExtractsDir); err != nil {
				return epdla.EndWithError(err)
			}
		}
	}

	epdla.EndWithResult("done")

	return nil
}

func extractPkgScripts(link vangogh_local_data.DownloadLink, dir string) error {
	return nil
}

func extractInnosetup(link vangogh_local_data.DownloadLink, dir string) error {
	return nil
}
