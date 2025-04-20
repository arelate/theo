package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func RemovePrefixHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	archive := !q.Has("no-archive")
	force := q.Has("force")

	return RemovePrefix(langCode, archive, force, ids...)
}

func RemovePrefix(langCode string, archive, force bool, ids ...string) error {

	rpa := nod.NewProgress("removing prefixes for %s...", strings.Join(ids, ","))
	defer rpa.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, vangogh_integration.SlugProperty)
	if err != nil {
		return err
	}

	rpa.TotalInt(len(ids))

	for _, id := range ids {
		if err := removeProductPrefix(id, langCode, rdx, archive, force); err != nil {
			return err
		}

		rpa.Increment()
	}

	return nil
}

func removeProductPrefix(id, langCode string, rdx redux.Readable, archive, force bool) error {
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

	if archive {
		if err = ArchivePrefix(langCode, id); err != nil {
			return err
		}
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
			if err = removeDirIfEmpty(absDir); err != nil {
				return err
			}
		}

		rpda.Increment()
	}

	return nil
}
