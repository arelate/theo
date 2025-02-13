package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path"
	"strings"
)

func SetPrefixEnvHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)

	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	var env []string
	if q.Has("env") {
		env = strings.Split(q.Get("env"), ",")
	}

	return SetPrefixEnv(langCode, env, ids...)
}

func SetPrefixEnv(langCode string, env []string, ids ...string) error {

	spea := nod.NewProgress("setting prefix environment variables for wine-run...")
	defer spea.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.SlugProperty, data.PrefixEnvProperty)
	if err != nil {
		return err
	}

	newEnvs := make(map[string][]string)

	spea.TotalInt(len(ids))

	for _, id := range ids {

		prefixName, err := data.GetPrefixName(id, rdx)
		if err != nil {
			return err
		}

		curEnv, _ := rdx.GetAllValues(data.PrefixEnvProperty, path.Join(prefixName, langCode))
		newEnvs[path.Join(prefixName, langCode)] = mergeEnv(curEnv, env)

		spea.Increment()
	}

	if err := rdx.BatchReplaceValues(data.PrefixEnvProperty, newEnvs); err != nil {
		return err
	}

	return nil
}

func mergeEnv(env1 []string, env2 []string) []string {
	de1, de2 := decodeEnv(env1), decodeEnv(env2)
	for k, v := range de2 {
		de1[k] = v
	}
	return encodeEnv(de1)
}

func decodeEnv(env []string) map[string]string {
	de := make(map[string]string, len(env))
	for _, e := range env {
		if k, v, ok := strings.Cut(e, "="); ok {
			de[k] = v
		}
	}
	return de
}

func encodeEnv(de map[string]string) []string {
	ee := make([]string, 0, len(de))
	for k, v := range de {
		ee = append(ee, k+"="+v)
	}
	return ee
}
