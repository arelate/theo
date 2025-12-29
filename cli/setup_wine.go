package cli

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

func SetupWineHandler(u *url.URL) error {

	q := u.Query()

	force := q.Has("force")

	return SetupWine(force)
}

func SetupWine(force bool) error {

	start := time.Now()

	currentOs := data.CurrentOs()

	if currentOs == vangogh_integration.Windows {
		err := errors.New("WINE is not required on Windows")
		return err
	}

	uwa := nod.Begin("setting up WINE for %s...", currentOs)
	defer uwa.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(),
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

	if err = validateWineBinaries(wbd, currentOs, start, force); err != nil {
		return err
	}

	if err = pinWineBinariesVersions(wbd, rdx); err != nil {
		return err
	}

	if err = cleanupDownloadedWineBinaries(wbd, currentOs); err != nil {
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

	req, err := data.ServerRequest(http.MethodGet, data.ApiWineBinariesVersions, nil, rdx)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
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

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem && wineBinary.OS != vangogh_integration.Windows {
			continue
		}

		if err := downloadWineBinary(&wineBinary, rdx, force); err != nil {
			return err
		}
	}

	return nil
}

func downloadWineBinary(binary *vangogh_integration.WineBinaryDetails, rdx redux.Readable, force bool) error {

	dwba := nod.NewProgress(" - %s %s...", binary.Title, binary.Version)
	defer dwba.Done()

	if err := rdx.MustHave(data.WineBinariesVersionsProperty, data.ServerConnectionProperties); err != nil {
		return err
	}

	wineDownloads := data.Pwd.AbsRelDirPath(data.WineDownloads, data.Wine)

	if currentVersion, ok := rdx.GetLastVal(data.WineBinariesVersionsProperty, binary.Title); ok && binary.Version == currentVersion && !force {
		dwba.EndWithResult("latest version already available")
		return nil
	}

	query := url.Values{
		"title": {binary.Title},
		"os":    {binary.OS.String()},
	}

	wineBinaryUrl, err := data.ServerUrl(data.HttpWineBinaryFilePath, query, rdx)
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if token, ok := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerSessionToken); ok && token != "" {
		dc.SetAuthorizationBearer(token)
	}

	return dc.Download(wineBinaryUrl, force, dwba, wineDownloads, binary.Filename)
}

func validateWineBinaries(wbd []vangogh_integration.WineBinaryDetails, operatingSystem vangogh_integration.OperatingSystem, since time.Time, force bool) error {

	vwba := nod.NewProgress("validating WINE binaries...")
	defer vwba.Done()

	wineDownloads := data.Pwd.AbsRelDirPath(data.WineDownloads, data.Wine)

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem && wineBinary.OS != vangogh_integration.Windows {
			continue
		}

		if err := wine_integration.ValidateWineBinary(&wineBinary, wineDownloads, since, force); err != nil {
			return err
		}
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

func cleanupDownloadedWineBinaries(wbd []vangogh_integration.WineBinaryDetails, operatingSystem vangogh_integration.OperatingSystem) error {

	cdwba := nod.NewProgress("cleaning up downloaded WINE binaries...")
	defer cdwba.Done()

	expectedFiles := make([]string, 0, len(wbd))
	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem && wineBinary.OS != vangogh_integration.Windows {
			continue
		}
		expectedFiles = append(expectedFiles, wineBinary.Filename)
	}

	wineDownloads := data.Pwd.AbsRelDirPath(data.WineDownloads, data.Wine)

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

	wineDownloads := data.Pwd.AbsRelDirPath(data.WineDownloads, data.Wine)
	wineBinaries := data.Pwd.AbsRelDirPath(data.WineBinaries, data.Wine)

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem {
			continue
		}

		srcPath := filepath.Join(wineDownloads, wineBinary.Filename)
		dstPath := filepath.Join(wineBinaries, pathways.Sanitize(wineBinary.Title), wineBinary.Version)

		if _, err := os.Stat(dstPath); err == nil && !force {
			continue
		}

		wba := nod.Begin(" - %s...", wineBinary.Title)

		if err := untar(srcPath, dstPath); err != nil {
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

	wineBinaries := data.Pwd.AbsRelDirPath(data.WineBinaries, data.Wine)

	absExpectedDirs := make([]string, 0)
	absActualDirs := make([]string, 0)

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem {
			continue
		}

		absTitleDir := filepath.Join(wineBinaries, pathways.Sanitize(wineBinary.Title))

		absLatestVersionDir := filepath.Join(absTitleDir, wineBinary.Version)
		absExpectedDirs = append(absExpectedDirs, absLatestVersionDir)

		titleDir, err := os.Open(absTitleDir)
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
		if err := os.RemoveAll(aud); err != nil {
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
