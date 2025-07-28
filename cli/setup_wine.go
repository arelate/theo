package cli

import (
	"encoding/json"
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/busan"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func SetupWineHandler(u *url.URL) error {

	q := u.Query()

	force := q.Has("force")

	return SetupWine(force)
}

func SetupWine(force bool) error {

	currentOs := data.CurrentOs()

	if currentOs == vangogh_integration.Windows {
		err := errors.New("WINE is not required on Windows")
		return err
	}

	uwa := nod.Begin("setting up WINE for %s...", currentOs)
	defer uwa.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir,
		data.ServerConnectionProperties,
		data.WineBinariesVersionsProperty)
	if err != nil {
		return err
	}

	wbd, err := getWineBinariesVersions(rdx)
	if err != nil {
		return err
	}

	if err = downloadWineBinaries(wbd, currentOs, rdx, force); err != nil {
		return err
	}

	if err = pinWineBinariesVersions(wbd, rdx); err != nil {
		return err
	}

	if err = cleanupDownloadedWineBinaries(wbd); err != nil {
		return err
	}

	if err = unpackWineBinaries(wbd, currentOs, force); err != nil {
		return err
	}

	if err = cleanupUnpackedWineBinaries(wbd, currentOs); err != nil {
		return err
	}

	if err = resetUmuConfigs(rdx); err != nil {
		return err
	}

	return nil
}

func getWineBinariesVersions(rdx redux.Readable) ([]vangogh_integration.WineBinaryDetails, error) {

	gwbva := nod.Begin("getting WINE binaries versions...")
	defer gwbva.Done()

	if err := rdx.MustHave(data.ServerConnectionProperties); err != nil {
		return nil, err
	}

	awbvu, err := data.ServerUrl(rdx, data.ApiWineBinariesVersions, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Get(awbvu.String())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New(resp.Status)
	}

	var wbd []vangogh_integration.WineBinaryDetails

	if err = json.NewDecoder(resp.Body).Decode(&wbd); err != nil {
		return nil, err
	}

	return wbd, nil
}

func downloadWineBinaries(wbd []vangogh_integration.WineBinaryDetails,
	operatingSystem vangogh_integration.OperatingSystem,
	rdx redux.Readable,
	force bool) error {

	dwba := nod.Begin("downloading WINE binaries...")
	defer dwba.Done()

	if err := rdx.MustHave(data.WineBinariesVersionsProperty); err != nil {
		return err
	}

	wineDownloads, err := pathways.GetAbsRelDir(data.WineDownloads)
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	for _, wineBinary := range wbd {

		if wineBinary.OS != operatingSystem && wineBinary.OS != vangogh_integration.Windows {
			continue
		}

		if currentVersion, ok := rdx.GetLastVal(data.WineBinariesVersionsProperty, wineBinary.Title); ok && wineBinary.Version == currentVersion && !force {
			continue
		}

		wba := nod.NewProgress(" - %s", wineBinary.Title)

		var wineBinaryUrl *url.URL
		params := map[string]string{
			"title": wineBinary.Title,
			"os":    wineBinary.OS.String(),
		}
		wineBinaryUrl, err = data.ServerUrl(rdx, data.HttpWineBinaryFilePath, params)
		if err != nil {
			return err
		}

		if err = dc.Download(wineBinaryUrl, force, wba, wineDownloads, wineBinary.Filename); err != nil {
			return err
		}

		wba.Done()
	}

	return nil
}

func pinWineBinariesVersions(wbd []vangogh_integration.WineBinaryDetails, rdx redux.Writeable) error {

	pwbva := nod.Begin("pinning WINE binaries versions...")
	defer pwbva.Done()

	if err := rdx.MustHave(data.WineBinariesVersionsProperty); err != nil {
		return err
	}

	wineBinariesVersions := make(map[string][]string)

	for _, wineBinary := range wbd {
		wineBinariesVersions[wineBinary.Title] = []string{wineBinary.Version}
	}

	return rdx.BatchReplaceValues(data.WineBinariesVersionsProperty, wineBinariesVersions)
}

func cleanupDownloadedWineBinaries(wbd []vangogh_integration.WineBinaryDetails) error {

	cdwba := nod.NewProgress("cleaning up downloaded WINE binaries...")
	defer cdwba.Done()

	expectedFiles := make([]string, 0, len(wbd))
	for _, wineBinary := range wbd {
		expectedFiles = append(expectedFiles, wineBinary.Filename)
	}

	wineDownloads, err := pathways.GetAbsRelDir(data.WineDownloads)
	if err != nil {
		return err
	}

	wineDownloadsDir, err := os.Open(wineDownloads)
	if err != nil {
		return err
	}

	defer wineDownloadsDir.Close()

	actualFiles, err := wineDownloadsDir.Readdirnames(-1)
	if err != nil {
		return err
	}

	unexpectedFiles := make([]string, 0)

	for _, af := range actualFiles {
		if strings.HasPrefix(af, ".") {
			continue
		}
		if !slices.Contains(expectedFiles, af) {
			unexpectedFiles = append(unexpectedFiles, af)
		}
	}

	if len(unexpectedFiles) == 0 {
		cdwba.EndWithResult("already clean")
		return nil
	}

	cdwba.TotalInt(len(unexpectedFiles))

	for _, uf := range unexpectedFiles {
		absUnexpectedFile := filepath.Join(wineDownloads, uf)
		if err = os.Remove(absUnexpectedFile); err != nil {
			return err
		}
		cdwba.Increment()
	}

	return nil
}

func unpackWineBinaries(wbd []vangogh_integration.WineBinaryDetails,
	operatingSystem vangogh_integration.OperatingSystem,
	force bool) error {

	uwba := nod.Begin("unpacking WINE binaries...")
	defer uwba.Done()

	wineDownloads, err := pathways.GetAbsRelDir(data.WineDownloads)
	if err != nil {
		return err
	}

	wineBinaries, err := pathways.GetAbsRelDir(data.WineBinaries)
	if err != nil {
		return err
	}

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem {
			continue
		}

		srcPath := filepath.Join(wineDownloads, wineBinary.Filename)
		dstPath := filepath.Join(wineBinaries, busan.Sanitize(wineBinary.Title), wineBinary.Version)

		if _, err = os.Stat(dstPath); err == nil && !force {
			continue
		}

		wba := nod.Begin(" - %s", wineBinary.Title)

		if err = untar(srcPath, dstPath); err != nil {
			return err
		}

		wba.Done()
	}

	return nil
}

func cleanupUnpackedWineBinaries(wbd []vangogh_integration.WineBinaryDetails,
	operatingSystem vangogh_integration.OperatingSystem) error {

	cuwba := nod.NewProgress("cleaning up unpacked WINE binaries...")
	defer cuwba.Done()

	wineBinaries, err := pathways.GetAbsRelDir(data.WineBinaries)
	if err != nil {
		return err
	}

	absExpectedDirs := make([]string, 0)
	absActualDirs := make([]string, 0)

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem {
			continue
		}

		absTitleDir := filepath.Join(wineBinaries, busan.Sanitize(wineBinary.Title))

		absLatestVersionDir := filepath.Join(absTitleDir, wineBinary.Version)
		absExpectedDirs = append(absExpectedDirs, absLatestVersionDir)

		var titleDir *os.File
		titleDir, err = os.Open(absTitleDir)
		if err != nil {
			return err
		}

		var filenames []string
		filenames, err = titleDir.Readdirnames(-1)
		if err != nil {
			if err = titleDir.Close(); err != nil {
				return err
			}
			return err
		}

		for _, fn := range filenames {
			absActualDirs = append(absActualDirs, filepath.Join(absTitleDir, fn))
		}

		if err = titleDir.Close(); err != nil {
			return err
		}
	}

	absUnexpectedDirs := make([]string, 0)

	for _, aad := range absActualDirs {
		if !slices.Contains(absExpectedDirs, aad) {
			absUnexpectedDirs = append(absUnexpectedDirs, aad)
		}
	}

	if len(absUnexpectedDirs) == 0 {
		cuwba.EndWithResult("already clean")
		return nil
	}

	cuwba.TotalInt(len(absUnexpectedDirs))

	for _, aud := range absUnexpectedDirs {
		if err = os.RemoveAll(aud); err != nil {
			return err
		}
		cuwba.Increment()
	}

	return nil
}

func untar(srcPath, dstPath string) error {

	if _, err := os.Stat(dstPath); err != nil {
		if err = os.MkdirAll(dstPath, 0755); err != nil {
			return err
		}
	}

	cmd := exec.Command("tar", "-xf", srcPath, "-C", dstPath)
	return cmd.Run()
}
