package cli

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/backups"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

var osEnvDefaults = map[vangogh_integration.OperatingSystem][]string{
	vangogh_integration.MacOS: {
		"CX_GRAPHICS_BACKEND=d3dmetal", // other values: dxmt, dxvk, wined3d
		"WINEMSYNC=1",
		"WINEESYNC=0",
		"ROSETTA_ADVERTISE_AVX=1",
		// "MTL_HUD_ENABLED=1", // not a candidate for default value, adding for reference
	},
}

func PrefixHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)

	// TODO: use installed lang-code for prefix
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}

	et := &execTask{
		exe:     q.Get("exe"),
		verbose: q.Has("verbose"),
	}

	if q.Has("env") {
		et.env = strings.Split(q.Get("env"), ",")
	}

	if q.Has("arg") {
		et.args = strings.Split(q.Get("arg"), ",")
	}

	mod := q.Get("mod")
	program := q.Get("program")

	defaultEnv := q.Has("default-env")
	deleteEnv := q.Has("delete-env")

	deleteExe := q.Has("delete-exe")

	deleteArg := q.Has("delete-arg")

	info := q.Has("info")
	archive := q.Has("archive")
	remove := q.Has("remove")
	force := q.Has("force")

	return Prefix(id, langCode,
		mod, program,
		defaultEnv, deleteEnv, deleteExe, deleteArg,
		info, archive, remove,
		et, force)
}

func Prefix(id string, langCode string,
	mod, program string,
	defaultEnv, deleteEnv, deleteExe, deleteArg bool,
	info, archive, remove bool,
	et *execTask, force bool) error {

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	et.prefix = absPrefixDir

	if deleteEnv {
		if err = prefixDeleteProperty(id, langCode, data.PrefixEnvProperty, rdx, force); err != nil {
			return err
		}
	}

	if defaultEnv {
		if err = prefixDefaultEnv(id, langCode, rdx); err != nil {
			return err
		}
	}

	if deleteExe {
		if err = prefixDeleteProperty(id, langCode, data.PrefixExeProperty, rdx, force); err != nil {
			return err
		}
	}

	if deleteArg {
		if err = prefixDeleteProperty(id, langCode, data.PrefixArgProperty, rdx, force); err != nil {
			return err
		}
	}

	if len(et.env) > 0 {
		if err = prefixSetEnv(id, langCode, et.env, rdx); err != nil {
			return err
		}
	}

	if et.exe != "" {
		if err = prefixSetExe(id, langCode, et.exe, rdx); err != nil {
			return err
		}
	}

	if len(et.args) > 0 {
		if err = prefixSetArgs(id, langCode, et.args, rdx); err != nil {
			return err
		}
	}

	if info {
		if err = prefixInfo(id, langCode, rdx); err != nil {
			return err
		}
	}

	if mod != "" {

		switch mod {
		case prefixModEnableRetina:
			if err = prefixModRetina(id, langCode, false, rdx, et.verbose, force); err != nil {
				return err
			}
		case prefixModDisableRetina:
			if err = prefixModRetina(id, langCode, true, rdx, et.verbose, force); err != nil {
				return err
			}
		}

	}

	if program != "" {

		if !slices.Contains(WinePrograms(), program) {
			return errors.New("unknown prefix WINE program " + program)
		}

		et.name = program
		et.exe = program

		if err = osExec(id, "", vangogh_integration.Windows, et, rdx, false); err != nil {
			return err
		}

	}

	if archive {
		if err = archiveProductPrefix(id, langCode); err != nil {
			return err
		}
	}

	if remove {
		if err = removeProductPrefix(id, langCode, rdx, force); err != nil {
			return err
		}
	}

	return nil
}

func archiveProductPrefix(id, langCode string) error {

	appa := nod.Begin("archiving prefix for %s...", id)
	defer appa.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir, vangogh_integration.SlugProperty)
	if err != nil {
		return err
	}

	prefixArchiveDir, err := pathways.GetAbsRelDir(data.PrefixArchive)
	if err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	absPrefixNameArchiveDir := filepath.Join(prefixArchiveDir, prefixName)

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absPrefixNameArchiveDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absPrefixNameArchiveDir, 0755); err != nil {
			return err
		}
	}

	if err = backups.Compress(absPrefixDir, absPrefixNameArchiveDir); err != nil {
		return err
	}

	return cleanupProductPrefixArchive(absPrefixNameArchiveDir)
}

func cleanupProductPrefixArchive(absPrefixNameArchiveDir string) error {
	cppa := nod.NewProgress(" cleaning up old prefix archives...")
	defer cppa.Done()

	return backups.Cleanup(absPrefixNameArchiveDir, true, cppa)
}

func prefixModRetina(id, langCode string, revert bool, rdx redux.Writeable, verbose, force bool) error {

	mpa := nod.Begin("modding retina in prefix for %s...", id)
	defer mpa.Done()

	if data.CurrentOs() != vangogh_integration.MacOS {
		mpa.EndWithResult("retina prefix mod is only applicable to %s", vangogh_integration.MacOS)
		return nil
	}

	if err := rdx.MustHave(vangogh_integration.SlugProperty, data.PrefixEnvProperty, data.PrefixExeProperty); err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	absDriveCroot := filepath.Join(absPrefixDir, relPrefixDriveCDir)

	regFilename := retinaOnFilename
	regContent := retinaOnReg
	if revert {
		regFilename = retinaOffFilename
		regContent = retinaOffReg
	}

	absRegPath := filepath.Join(absDriveCroot, regFilename)
	if _, err = os.Stat(absRegPath); os.IsNotExist(err) || (err == nil && force) {
		if err = createRegFile(absRegPath, regContent); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	et := &execTask{
		exe:     regeditBin,
		workDir: absDriveCroot,
		args:    []string{absRegPath},
		verbose: verbose,
	}

	switch data.CurrentOs() {
	case vangogh_integration.MacOS:
		if err := macOsWineRun(id, langCode, rdx, et, force); err != nil {
			return err
		}
	default:
		// do nothing
		return nil
	}
	return nil
}

func createRegFile(absPath string, content []byte) error {

	regFile, err := os.Create(absPath)
	if err != nil {
		return err
	}
	defer regFile.Close()

	if _, err := io.Copy(regFile, bytes.NewReader(content)); err != nil {
		return err
	}

	return nil
}

func removeProductPrefix(id, langCode string, rdx redux.Readable, force bool) error {
	rppa := nod.Begin(" removing installed files from prefix for %s...", id)
	defer rppa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absPrefixDir); os.IsNotExist(err) {
		rppa.EndWithResult("not present")
		return nil
	}

	if !force {
		rppa.EndWithResult("found prefix, use -force to remove")
		return nil
	}

	relInventoryFiles, err := readInventory(id, langCode, vangogh_integration.Windows, rdx)
	if os.IsNotExist(err) {
		rppa.EndWithResult("installed files inventory not found")
		return nil
	} else if err != nil {
		return err
	}

	if err = removePrefixInstalledFiles(absPrefixDir, relInventoryFiles...); err != nil {
		return err
	}

	if err = removePrefixDirs(absPrefixDir, relInventoryFiles...); err != nil {
		return err
	}

	return nil
}

func removePrefixInstalledFiles(absPrefixDir string, relFiles ...string) error {
	rpifa := nod.NewProgress(" removing inventoried files in prefix...")
	defer rpifa.Done()

	rpifa.TotalInt(len(relFiles))

	for _, relFile := range relFiles {

		absInventoryFile := filepath.Join(absPrefixDir, relFile)
		if stat, err := os.Stat(absInventoryFile); err == nil && !stat.IsDir() {
			if err = os.Remove(absInventoryFile); err != nil {
				return err
			}
		}

		rpifa.Increment()
	}

	return nil
}

func removePrefixDirs(absPrefixDir string, relFiles ...string) error {
	rpda := nod.NewProgress(" removing prefix empty directories...")
	defer rpda.Done()

	rpda.TotalInt(len(relFiles))

	// filepath.Walk adds files in lexical order and for removal we want to reverse that to attempt to remove
	// leafs first, roots last
	slices.Reverse(relFiles)

	for _, relFile := range relFiles {

		absDir := filepath.Join(absPrefixDir, relFile)
		if stat, err := os.Stat(absDir); err == nil && stat.IsDir() {
			var empty bool
			if empty, err = osIsDirEmpty(absDir); empty && err == nil {
				if err = os.RemoveAll(absDir); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}

		rpda.Increment()
	}

	return nil
}

func prefixSetEnv(id, langCode string, env []string, rdx redux.Writeable) error {

	spea := nod.Begin("setting %s...", data.PrefixEnvProperty)
	defer spea.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty, data.PrefixEnvProperty); err != nil {
		return err
	}

	newEnvs := make(map[string][]string)

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	curEnv, _ := rdx.GetAllValues(data.PrefixEnvProperty, path.Join(prefixName, langCode))
	newEnvs[path.Join(prefixName, langCode)] = mergeEnv(curEnv, env)

	if err = rdx.BatchReplaceValues(data.PrefixEnvProperty, newEnvs); err != nil {
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

func prefixSetExe(id, langCode string, exe string, rdx redux.Writeable) error {

	spepa := nod.Begin("setting %s...", data.PrefixExeProperty)
	defer spepa.Done()

	if strings.HasPrefix(exe, ".") ||
		strings.HasPrefix(exe, "/") {
		spepa.EndWithResult("exe path must be relative and cannot start with . or /")
		return nil
	}

	if err := rdx.MustHave(vangogh_integration.SlugProperty, data.PrefixExeProperty); err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	absExePath := filepath.Join(absPrefixDir, relPrefixDriveCDir, exe)
	if _, err = os.Stat(absExePath); err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	langPrefixName := path.Join(prefixName, langCode)

	return rdx.ReplaceValues(data.PrefixExeProperty, langPrefixName, exe)
}

func prefixSetArgs(id, langCode string, args []string, rdx redux.Writeable) error {

	spepa := nod.Begin("setting %s...", data.PrefixArgProperty)
	defer spepa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty, data.PrefixArgProperty); err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	langPrefixName := path.Join(prefixName, langCode)

	return rdx.ReplaceValues(data.PrefixArgProperty, langPrefixName, args...)
}

func prefixInfo(id, langCode string, rdx redux.Readable) error {

	pia := nod.Begin("looking up prefix details...")
	defer pia.Done()

	if err := rdx.MustHave(vangogh_integration.TitleProperty,
		data.PrefixEnvProperty,
		data.PrefixExeProperty); err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}
	langPrefixName := path.Join(prefixName, langCode)

	summary := make(map[string][]string)

	properties := []string{data.PrefixEnvProperty, data.PrefixExeProperty, data.PrefixArgProperty}

	for _, p := range properties {
		if values, ok := rdx.GetAllValues(p, langPrefixName); ok {
			for _, value := range values {
				summary[langPrefixName] = append(summary[langPrefixName], fmt.Sprintf("%s:%s", p, value))
			}
		}
	}

	if len(summary) == 0 {
		pia.EndWithResult("found nothing")
	} else {
		pia.EndWithSummary("results:", summary)
	}

	return nil
}

func prefixDefaultEnv(id, langCode string, rdx redux.Writeable) error {

	pdea := nod.Begin("defaulting prefix environment variables...")
	defer pdea.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty, data.PrefixEnvProperty); err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	langPrefixName := path.Join(prefixName, langCode)

	return rdx.ReplaceValues(data.PrefixEnvProperty, langPrefixName, osEnvDefaults[data.CurrentOs()]...)
}

func prefixDeleteProperty(id, langCode, property string, rdx redux.Writeable, force bool) error {
	pdea := nod.Begin("deleting %s...", property)
	defer pdea.Done()

	if !force {
		pdea.EndWithResult("this operation requires -force flag")
		return nil
	}

	if err := rdx.MustHave(vangogh_integration.SlugProperty, property); err != nil {
		return err
	}

	prefixName, err := data.GetPrefixName(id, rdx)
	if err != nil {
		return err
	}

	langPrefixName := path.Join(prefixName, langCode)

	return rdx.CutKeys(property, langPrefixName)
}
