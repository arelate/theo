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

func getDownloadsManifest(id string, rdx redux.Writeable, force bool) (*vangogh_integration.DownloadsManifest, error) {

	gtma := nod.NewProgress(" getting downloads manifest...")
	defer gtma.Done()

	downloadsManifestsDir, err := pathways.GetAbsRelDir(data.DownloadsManifests)
	if err != nil {
		return nil, err
	}

	kvDownloadsManifests, err := kevlar.New(downloadsManifestsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if dm, err := readLocalDownloadsManifest(id, kvDownloadsManifests); err != nil {
		return nil, err
	} else if dm != nil && !force {
		gtma.EndWithResult("read local")
		return dm, nil
	}

	if err = rdx.MustHave(data.ServerConnectionProperties, data.TitleProperty, data.SlugProperty); err != nil {
		return nil, err
	}

	defer gtma.EndWithResult("fetched remote")
	if dm, err := fetchRemoteDownloadsManifest(id, rdx, kvDownloadsManifests); err != nil {
		return nil, err
	} else {

		if err = rdx.ReplaceValues(data.TitleProperty, id, dm.Title); err != nil {
			return nil, err
		}
		if err = rdx.ReplaceValues(data.SlugProperty, id, dm.Slug); err != nil {
			return nil, err
		}

		return dm, nil
	}
}

func readLocalDownloadsManifest(id string, kvDownloadsManifests kevlar.KeyValues) (*vangogh_integration.DownloadsManifest, error) {

	if has := kvDownloadsManifests.Has(id); !has {
		return nil, nil
	}

	tmReadCloser, err := kvDownloadsManifests.Get(id)
	if err != nil {
		return nil, err
	}
	defer tmReadCloser.Close()

	var dm vangogh_integration.DownloadsManifest
	if err = json.NewDecoder(tmReadCloser).Decode(&dm); err != nil {
		return nil, err
	}

	return &dm, nil
}

func fetchRemoteDownloadsManifest(id string, rdx redux.Readable, kvDownloadsManifests kevlar.KeyValues) (*vangogh_integration.DownloadsManifest, error) {

	vdmu, err := data.ServerUrl(rdx,
		data.ServerDownloadsManifestPath,
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
		return nil, errors.New("error fetching downloads manifest: " + resp.Status)
	}

	var bts []byte
	buf := bytes.NewBuffer(bts)
	tr := io.TeeReader(resp.Body, buf)

	if err := kvDownloadsManifests.Set(id, tr); err != nil {
		return nil, err
	}

	var dm vangogh_integration.DownloadsManifest
	if err = json.NewDecoder(buf).Decode(&dm); err != nil {
		return nil, err
	}

	return &dm, nil
}
