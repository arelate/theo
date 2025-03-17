package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"net/http"
)

func getTheoMetadata(id string, rdx redux.Writeable, force bool) (*vangogh_integration.TheoMetadata, error) {

	gtma := nod.NewProgress(" getting theo metadata...")
	defer gtma.Done()

	theoMetadataDir, err := pathways.GetAbsRelDir(data.TheoMetadata)
	if err != nil {
		return nil, err
	}

	kvTheoMetadata, err := kevlar.New(theoMetadataDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if tm, err := readLocalTheoMetadata(id, kvTheoMetadata); err != nil {
		return nil, err
	} else if tm != nil && !force {
		gtma.EndWithResult("read local")
		return tm, nil
	}

	if err = rdx.MustHave(data.ServerConnectionProperties, data.TitleProperty, data.SlugProperty); err != nil {
		return nil, err
	}

	defer gtma.EndWithResult("fetched remote")
	if tm, err := fetchRemoteTheoMetadata(id, rdx, kvTheoMetadata); err != nil {
		return nil, err
	} else {

		if err = rdx.ReplaceValues(data.TitleProperty, id, tm.Title); err != nil {
			return nil, err
		}
		if err = rdx.ReplaceValues(data.SlugProperty, id, tm.Slug); err != nil {
			return nil, err
		}

		return tm, nil
	}
}

func readLocalTheoMetadata(id string, kvTheoMetadata kevlar.KeyValues) (*vangogh_integration.TheoMetadata, error) {

	if has := kvTheoMetadata.Has(id); !has {
		return nil, nil
	}

	tmReadCloser, err := kvTheoMetadata.Get(id)
	if err != nil {
		return nil, err
	}
	defer tmReadCloser.Close()

	var tm vangogh_integration.TheoMetadata
	if err := json.NewDecoder(tmReadCloser).Decode(&tm); err != nil {
		return nil, err
	}

	return &tm, nil
}

func fetchRemoteTheoMetadata(id string, rdx redux.Readable, kvTheoMetadata kevlar.KeyValues) (*vangogh_integration.TheoMetadata, error) {

	vdmu, err := data.ServerUrl(rdx,
		data.ServerTheoMetadataPath,
		map[string]string{vangogh_integration.IdProperty: id})
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Get(vdmu.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New("error fetching theo metadata: " + resp.Status)
	}

	var bts []byte
	buf := bytes.NewBuffer(bts)
	tr := io.TeeReader(resp.Body, buf)

	if err := kvTheoMetadata.Set(id, tr); err != nil {
		return nil, err
	}

	var tm vangogh_integration.TheoMetadata
	if err := json.NewDecoder(buf).Decode(&tm); err != nil {
		return nil, err
	}

	return &tm, nil
}
