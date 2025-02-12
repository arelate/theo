package cli

import (
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
)

func DeletePrefixEnvHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	force := q.Has("force")

	return DeletePrefixEnv(ids, force)
}

func DeletePrefixEnv(ids []string, force bool) error {

	dpea := nod.Begin("deleting prefix environment variables...")
	defer dpea.Done()

	if !force {
		dpea.EndWithResult("this operation requires force flag")
		return nil
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.PrefixEnvProperty)
	if err != nil {
		return err
	}

	prefixes := make([]string, 0, len(ids))
	for _, id := range ids {
		prefixName, err := data.GetPrefixName(id, rdx)
		if err != nil {
			return err
		}

		prefixes = append(prefixes, prefixName)
	}

	if err = rdx.CutKeys(data.PrefixEnvProperty, prefixes...); err != nil {
		return err
	}

	return nil
}
