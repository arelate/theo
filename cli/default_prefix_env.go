package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path"
)

var operatingSystemEnvDefaults = map[vangogh_integration.OperatingSystem][]string{
	vangogh_integration.MacOS: {
		"WINED3DMETAL=1",
		"WINEDXVK=0",
		"WINEMSYNC=1",
		"WINEESYNC=0",
		"ROSETTA_ADVERTISE_AVX=1",
	},
}

func DefaultPrefixEnvHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)

	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	return DefaultPrefixEnv(langCode, ids...)
}

func DefaultPrefixEnv(langCode string, ids ...string) error {

	dpea := nod.Begin("setting prefix environment variables to defaults...")
	defer dpea.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.PrefixEnvProperty)
	if err != nil {
		return err
	}

	defaultEnvs := make(map[string][]string, len(ids))
	for _, id := range ids {
		prefixName, err := data.GetPrefixName(id, rdx)
		if err != nil {
			return err
		}

		defaultEnvs[path.Join(prefixName, langCode)] = operatingSystemEnvDefaults[data.CurrentOs()]
	}

	if err = rdx.BatchReplaceValues(data.PrefixEnvProperty, defaultEnvs); err != nil {
		return err

	}

	return nil
}
