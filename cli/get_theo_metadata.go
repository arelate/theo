package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/http"
	"net/url"
)

func GetTheoMetadataHandler(u *url.URL) error {

	ids := Ids(u)
	force := u.Query().Has("force")

	return GetTheoMetadata(ids, force)
}

func GetTheoMetadata(ids []string, force bool) error {

	gdma := nod.NewProgress("getting theo metadata...")
	defer gdma.EndWithResult("done")

	gdma.TotalInt(len(ids))

	theoMetadataDir, err := pathways.GetAbsRelDir(data.TheoMetadata)
	if err != nil {
		return gdma.EndWithError(err)
	}

	kvTheoMetadata, err := kevlar.NewKeyValues(theoMetadataDir, kevlar.JsonExt)
	if err != nil {
		return gdma.EndWithError(err)
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return gdma.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(reduxDir, data.SetupProperties)
	if err != nil {
		return gdma.EndWithError(err)
	}

	for _, id := range ids {

		if err = getProductTheoMetadata(id, rdx, kvTheoMetadata, force); err != nil {
			return gdma.EndWithError(err)
		}

		gdma.Increment()
	}

	return nil
}

func getProductTheoMetadata(id string, rdx kevlar.ReadableRedux, kvTheoMetadata kevlar.KeyValues, force bool) error {

	if has, err := kvTheoMetadata.Has(id); err == nil {
		if has && !force {
			return nil
		}
	} else {
		return err
	}

	vdmu, err := data.VangoghUrl(rdx,
		data.VangoghTheoMetadataPath,
		map[string]string{vangogh_local_data.IdProperty: id})
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Get(vdmu.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	return kvTheoMetadata.Set(id, resp.Body)
}
