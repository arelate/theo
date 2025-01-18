package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"net/url"
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

	return SetPrefixEnv(ids, langCode, env)
}

func SetPrefixEnv(ids []string, langCode string, env []string) error {

	spea := nod.NewProgress("setting prefix environment variables that will be used with wine-run...")
	defer spea.EndWithResult("done")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return spea.EndWithError(err)
	}

	rdx, err := kevlar.NewReduxWriter(reduxDir, data.PrefixEnvProperty)
	if err != nil {
		return spea.EndWithError(err)
	}

	newEnvs := make(map[string][]string)

	spea.TotalInt(len(ids))

	for _, id := range ids {
		prefixName := data.GetPrefixName(id, langCode)
		curEnv, _ := rdx.GetAllValues(data.PrefixEnvProperty, prefixName)
		newEnvs[prefixName] = mergeEnv(curEnv, env)

		spea.Increment()
	}

	if err := rdx.BatchReplaceValues(data.PrefixEnvProperty, newEnvs); err != nil {
		return spea.EndWithError(err)
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
