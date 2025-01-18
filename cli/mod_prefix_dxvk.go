package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"io"
	"net/url"
	"os"
	"path/filepath"
)

const (
	dxVkDlls64Glob = "x64/*.dll"
	dxVkDlls32Glob = "x32/*.dll"
)

const (
	pfxSystem64Path = "windows/system32"
	pfxSystem32Path = "windows/syswow64"
)

const originalExt = ".original"

func ModPrefixDxVkHandler(u *url.URL) error {

	q := u.Query()

	id := q.Get(vangogh_integration.IdProperty)
	langCode := defaultLangCode
	if q.Has(vangogh_integration.LanguageCodeProperty) {
		langCode = q.Get(vangogh_integration.LanguageCodeProperty)
	}
	dxVkRepo := q.Get("dxvk-repo")
	revert := q.Has("revert")

	return ModPrefixDxVk(id, langCode, dxVkRepo, revert)
}

func ModPrefixDxVk(id, langCode string, dxVkRepo string, revert bool) error {
	mpa := nod.Begin("modding DXVK in prefix for %s...", id)

	msg := "enabled, use `-env WINEDLLOVERRIDES=\"d3d10core,d3d11=n,b;\"` when running apps in that prefix"
	if revert {
		msg = "disabled"
	}
	defer mpa.EndWithResult(msg)

	if data.CurrentOS() != vangogh_integration.MacOS {
		mpa.EndWithResult("DXVK prefix mod is only applicable to macOS")
		return nil
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode)
	if err != nil {
		return mpa.EndWithError(err)
	}

	dxVkSource, release, err := data.GetDxVkSourceLatestRelease(dxVkRepo)
	if err != nil {
		return mpa.EndWithError(err)
	}

	binariesDir, err := pathways.GetAbsRelDir(data.Binaries)
	if err != nil {
		return mpa.EndWithError(err)
	}

	absBinaryDir := filepath.Join(binariesDir, dxVkSource.Owner, dxVkSource.Repo, busan.Sanitize(release.TagName))

	return toggleDxVk(absPrefixDir, absBinaryDir, revert)
}

func toggleDxVk(absPrefixDir, absAssetDir string, revert bool) error {

	if data.CurrentOS() != vangogh_integration.MacOS {
		return nil
	}

	// identify x32/x64 release binaries for DXVK-macos (abs paths)
	// locate target abs path in the prefix
	// copy originals to allow restoration
	// based on the flag copy over / restore .dll files

	var dxVkFn func(absAssetSrcDir, srcGlob, absPrefixDir, relDstDir string) error

	if revert {
		dxVkFn = disableDxVk
	} else {
		dxVkFn = enableDxVk
	}

	if err := dxVkFn(absAssetDir, dxVkDlls64Glob, absPrefixDir, pfxSystem64Path); err != nil {
		return err
	}

	if err := dxVkFn(absAssetDir, dxVkDlls32Glob, absPrefixDir, pfxSystem32Path); err != nil {
		return err
	}

	return nil
}

func disableDxVk(absAssetSrcDir, srcGlob, absPrefixDir, relDstDir string) error {

	dda := nod.Begin(" reverting DXVK...")
	defer dda.EndWithResult("done")

	pattern := filepath.Join(absAssetSrcDir, "*", srcGlob)
	dxVkSrcMatches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	absTargetDir := filepath.Join(absPrefixDir, relPrefixDriveCDir, relDstDir)

	for _, match := range dxVkSrcMatches {

		_, filename := filepath.Split(match)
		absTargetDstPath := filepath.Join(absTargetDir, filename)

		if err := restoreOriginalFile(absTargetDstPath); os.IsNotExist(err) {
			dda.EndWithResult("originals backup not found, DXVK was not enabled in this prefix")
			return nil
		} else if err != nil {
			return err
		}
	}

	return nil
}

func enableDxVk(absAssetSrcDir, srcGlob, absPrefixDir, relDstDir string) error {

	eda := nod.Begin(" enabling DXVK...")
	defer eda.EndWithResult("done")

	pattern := filepath.Join(absAssetSrcDir, "*", srcGlob)
	dxVkSrcMatches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	absTargetDir := filepath.Join(absPrefixDir, relPrefixDriveCDir, relDstDir)

	for _, match := range dxVkSrcMatches {

		_, filename := filepath.Split(match)
		absTargetDstPath := filepath.Join(absTargetDir, filename)

		if err := backupOriginalFile(absTargetDstPath); os.IsExist(err) {
			eda.EndWithResult("found originals backup, DXVK was already enabled in this prefix")
			return nil
		} else if err != nil {
			return err
		}

		if err := copyFile(match, absTargetDstPath); err != nil {
			return err
		}
	}

	return nil
}

func backupOriginalFile(absPath string) error {

	absBackupPath := absPath + originalExt

	if _, err := os.Stat(absBackupPath); err == nil {
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.Rename(absPath, absBackupPath)
}

func restoreOriginalFile(absPath string) error {
	absBackupPath := absPath + originalExt

	if _, err := os.Stat(absBackupPath); err != nil {
		return err
	}

	if err := os.Remove(absPath); err != nil {
		return err
	}

	return os.Rename(absBackupPath, absPath)
}

func copyFile(src, dst string) error {

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}
