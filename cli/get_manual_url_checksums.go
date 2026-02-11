package cli

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func getManualUrlChecksums(id string, rdx redux.Writeable, force bool) (map[string]string, error) {

	gmuca := nod.NewProgress(" getting manual-url checksums for %s...", id)
	defer gmuca.Done()

	manualUrlChecksumsDir := data.Pwd.AbsRelDirPath(data.ManualUrlChecksums, vangogh_integration.Metadata)

	kvManualUrlChecksums, err := kevlar.New(manualUrlChecksumsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	if dm, err := readLocalManualUrlChecksums(id, kvManualUrlChecksums); err != nil {
		return nil, err
	} else if dm != nil && !force {
		gmuca.EndWithResult("read local")
		return dm, nil
	}

	if err = vangoghValidateSessionToken(rdx); err != nil {
		return nil, err
	}

	manualUrlChecksums, err := fetchRemoteManualUrlChecksums(id, rdx, kvManualUrlChecksums)
	if err != nil {
		return nil, err
	}

	gmuca.EndWithResult("fetched remote")

	return manualUrlChecksums, nil
}

func readLocalManualUrlChecksums(id string, kvManualUrlChecksums kevlar.KeyValues) (map[string]string, error) {

	if has := kvManualUrlChecksums.Has(id); !has {
		return nil, nil
	}

	tmReadCloser, err := kvManualUrlChecksums.Get(id)
	if err != nil {
		return nil, err
	}
	defer tmReadCloser.Close()

	var manualUrlChecksums map[string]string
	if err = json.UnmarshalRead(tmReadCloser, &manualUrlChecksums); err != nil {
		return nil, err
	}

	return manualUrlChecksums, nil
}

func fetchRemoteManualUrlChecksums(id string, rdx redux.Readable, kvManualUrlChecksums kevlar.KeyValues) (map[string]string, error) {

	fra := nod.Begin(" fetching remote manual-url checksums for %s...", id)
	defer fra.Done()

	query := url.Values{
		vangogh_integration.IdProperty: {id},
	}

	req, err := data.VangoghRequest(http.MethodGet, data.ApiManualUrlChecksums, query, rdx)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New("error fetching manual-url checksums: " + resp.Status)
	}

	var bts []byte
	buf := bytes.NewBuffer(bts)
	tr := io.TeeReader(resp.Body, buf)

	if err = kvManualUrlChecksums.Set(id, tr); err != nil {
		return nil, err
	}

	var manualUrlChecksums map[string]string
	if err = json.UnmarshalRead(buf, &manualUrlChecksums); err != nil {
		return nil, err
	}

	return manualUrlChecksums, nil
}
