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

func GetDownloadsMetadataHandler(u *url.URL) error {

	ids := Ids(u)
	force := u.Query().Has("force")

	return GetDownloadsMetadata(ids, force)
}

func GetDownloadsMetadata(ids []string, force bool) error {

	gdma := nod.NewProgress("getting downloads metadata...")
	defer gdma.End()

	gdma.TotalInt(len(ids))

	dmd, err := pathways.GetAbsRelDir(data.DownloadsMetadata)
	if err != nil {
		return gdma.EndWithError(err)
	}

	kvdm, err := kevlar.NewKeyValues(dmd, kevlar.JsonExt)
	if err != nil {
		return gdma.EndWithError(err)
	}

	rdp, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return gdma.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxReader(rdp, data.SetupProperties)
	if err != nil {
		return gdma.EndWithError(err)
	}

	for _, id := range ids {

		if err = getProductDownloadsMetadata(id, rdx, kvdm, force); err != nil {
			return gdma.EndWithError(err)
		}

		gdma.Increment()
	}

	gdma.EndWithResult("done")

	return nil
}

func getProductDownloadsMetadata(id string, rdx kevlar.ReadableRedux, kv kevlar.KeyValues, force bool) error {

	if has, err := kv.Has(id); err == nil {
		if has && !force {
			return nil
		}
	} else {
		return err
	}

	vdmu, err := data.VangoghUrl(rdx,
		data.VangoghDownloadsMetadataPath,
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

	return kv.Set(id, resp.Body)
}
