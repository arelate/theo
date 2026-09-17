package cli

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func egsRemoveChunks(appName string, operatingSystem vangogh_integration.OperatingSystem, originData *data.OriginData) error {

	erca := nod.NewProgress(" removing EGS chunks...")
	defer erca.Done()

	erca.TotalInt(len(originData.Manifest.ChunkList.Chunks))

	featureLevel := originData.Manifest.Metadata.FeatureLevel
	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, operatingSystem)

	for _, chunk := range originData.Manifest.ChunkList.Chunks {
		absChunkPath := filepath.Join(absChunksDownloadsDir, chunk.Path(featureLevel))
		if _, err := os.Stat(absChunkPath); os.IsNotExist(err) {
			erca.Increment()
			continue
		}
		if err := os.Remove(absChunkPath); err != nil {
			return err
		}
		erca.Increment()
	}

	return os.RemoveAll(absChunksDownloadsDir)
}

func egsUninstall(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	eua := nod.NewProgress("uninstalling EGS %s...", appName)
	defer eua.Done()

	eua.TotalInt(len(originData.Manifest.FileList.List))

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return err
	}

	for _, file := range originData.Manifest.FileList.List {
		absFilePath := filepath.Join(installedPath, file.Filename)
		if _, err = os.Stat(absFilePath); os.IsNotExist(err) {
			eua.Increment()
			continue
		}
		if err = os.Remove(absFilePath); err != nil {
			return err
		}
	}

	var installedFiles []string
	if installedFiles, err = relWalkDir(installedPath); err == nil && len(installedFiles) == 0 {
		if err = os.RemoveAll(installedPath); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func egsAssembleChunks(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	eaca := nod.NewProgress("assembling EGS chunks into files for %s-%s...", appName, ii.OperatingSystem)
	defer eaca.Done()

	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, ii.OperatingSystem)

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return err
	}

	eaca.Total(uint64(egsManifestSize(originData.Manifest)))

	for _, chunkedFile := range originData.Manifest.FileList.List {
		if err = egsAssembleFile(&chunkedFile, originData.Manifest.Metadata.FeatureLevel, absChunksDownloadsDir, installedPath); err != nil {
			return err
		}

		eaca.Progress(chunkedFile.Size)
	}

	return nil
}

func egsAssembleFile(chunkedFile *egs_integration.File, featureLevel uint32, chunksDir, installedPath string) error {

	var err error

	absOutputFilename := filepath.Join(installedPath, chunkedFile.Filename)
	absOutputDir, _ := filepath.Split(absOutputFilename)

	if _, err = os.Stat(absOutputDir); os.IsNotExist(err) {
		if err = os.MkdirAll(absOutputDir, camino.DefaultFileMode); err != nil {
			return err
		}
	}

	outFile, err := os.Create(absOutputFilename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for _, part := range chunkedFile.Parts {

		if err = egsWriteChunkPart(&part, featureLevel, chunksDir, outFile); err != nil {
			return err
		}
	}

	return nil
}

func egsWriteChunkPart(part *egs_integration.ChunkPart, featureLevel uint32, chunksDir string, outFile *os.File) error {

	chunkPath := filepath.Join(chunksDir, part.Chunk.Path(featureLevel))

	var chunkFile *os.File
	chunkFile, err := os.Open(chunkPath)
	if err != nil {
		return err
	}
	defer chunkFile.Close()

	var chunkReader io.Reader
	chunkReader, err = egs_integration.ReadChunk(chunkFile)
	if err != nil {
		return nil
	}

	var chunkData []byte
	chunkData, err = io.ReadAll(chunkReader)
	if err != nil {
		return err
	}

	if _, err = io.Copy(outFile, bytes.NewReader(chunkData[part.Offset:part.Offset+part.Size])); err != nil {
		return err
	}

	return nil
}

func egsValidateAssembly(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	evaa := nod.NewProgress("validating assembled files for %s-%s...", appName, ii.Origin)
	defer evaa.Done()

	evaa.Total(uint64(egsManifestSize(originData.Manifest)))

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return err
	}

	for _, file := range originData.Manifest.FileList.List {
		if err = egsValidateAssembledFile(installedPath, &file); err != nil {
			return err
		}

		evaa.Progress(file.Size)
	}

	return nil
}

func egsValidateAssembledFile(installedDir string, assembledFile *egs_integration.File) error {

	var err error

	absFilename := filepath.Join(installedDir, assembledFile.Filename)

	inputFile, err := os.Open(absFilename)
	if err != nil {
		return err
	}

	shaSum := sha1.New()

	if _, err = io.Copy(shaSum, inputFile); err != nil {
		return err
	}

	actualShaSum := fmt.Sprintf("%x", shaSum.Sum(nil))
	expectedShaSum := fmt.Sprintf("%x", assembledFile.ShaHash)

	if actualShaSum != expectedShaSum {
		return errors.New("failed validation for " + assembledFile.Filename)
	}

	return nil
}

func egsChmodLauncherExe(id string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	switch ii.OperatingSystem {

	case vangogh_integration.MacOS:

		installedPath, err := originOsInstalledPath(id, ii, rdx)
		if err != nil {
			return err
		}

		manifestLaunchExe := originData.Manifest.Metadata.LaunchExe

		absLaunchExePath := filepath.Join(installedPath, manifestLaunchExe)

		if _, err = os.Stat(absLaunchExePath); err == nil {
			if err = chmodExecutable(absLaunchExePath); err != nil {
				return err
			}
		}
	default:
		// do nothing
	}

	return nil
}

func egsManifestVersion(manifest *egs_integration.Manifest) string {
	if manifest != nil &&
		manifest.Metadata != nil {
		return manifest.Metadata.BuildVersion
	}
	return ""
}

func egsManifestSize(manifest *egs_integration.Manifest) int64 {
	var totalEstimatedBytes int64

	for _, file := range manifest.FileList.List {
		totalEstimatedBytes += int64(file.Size)
	}

	return totalEstimatedBytes
}

func egsAssembleValidateChunks(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Readable) error {

	egsAppsDir := camino.GetRel(vangogh_integration.EgsApps, vangogh_integration.InstalledApps)

	if err := originHasFreeSpace(appName, egsAppsDir, ii, originData); err != nil {
		return err
	}

	if err := egsAssembleChunks(appName, ii, originData, rdx); err != nil {
		return err
	}

	if err := egsValidateAssembly(appName, ii, originData, rdx); err != nil {
		return err
	}

	return nil
}

func egsInstallDownloadableContent(ii *InstallInfo, catalogItem *egs_integration.CatalogItem) error {

	if len(catalogItem.DlcItemList) == 0 {
		return nil
	}

	eidca := nod.Begin("installing DLCs for %s...", catalogItem.Title)
	defer eidca.Done()

	osGameAssets, err := egs_integration.GetGameAssets(ii.force)
	if err != nil {
		return err
	}

	dlcGameAssets, err := egs_integration.CatalogItemDlcGameAssets(osGameAssets, ii.OperatingSystem, catalogItem, ii.force)
	if err != nil {
		return err
	}

	for dlcAppName, dlcTitle := range dlcGameAssets {
		if err = Install(dlcAppName, ii); err != nil {
			return err
		}

		ii.DownloadableContent = append(ii.DownloadableContent, dlcTitle)
	}

	return nil
}

func egsUninstallDlcs(appName string, ii *InstallInfo, rdx redux.Writeable) error {

	eudca := nod.Begin("uninstalling DLCs for %s...", appName)
	defer eudca.Done()

	gameAsset, err := egs_integration.GetGameAsset(appName, ii.OperatingSystem, ii.force)
	if err != nil {
		return err
	}

	catalogItem, err := egs_integration.GetCatalogItem(gameAsset, rdx, ii.force)
	if err != nil {
		return err
	}

	osGameAssets, err := egs_integration.GetGameAssets(ii.force)
	if err != nil {
		return err
	}

	catalogItemDlcs, err := egs_integration.CatalogItemDlcGameAssets(osGameAssets, ii.OperatingSystem, catalogItem, ii.force)
	for dlcItemId := range catalogItemDlcs {
		if err = originUninstall(dlcItemId, ii, rdx); err != nil {
			return err
		}
	}

	return nil
}

func egsValidateChunks(appName string, ii *InstallInfo, originData *data.OriginData) error {

	evca := nod.NewProgress("validating EGS chunks for %s-%s...", appName, ii.OperatingSystem)
	defer evca.Done()

	evca.Total(uint64(egsManifestSize(originData.Manifest)))

	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, ii.OperatingSystem)

	for _, chunk := range originData.Manifest.ChunkList.Chunks {

		chunkPath := chunk.Path(originData.Manifest.Metadata.FeatureLevel)

		absChunkFilename := filepath.Join(absChunksDownloadsDir, chunkPath)

		chunkFile, err := os.Open(absChunkFilename)
		if err != nil {
			return err
		}

		chunkReader, err := egs_integration.ReadChunk(chunkFile)
		if err != nil {
			return err
		}

		shaSum := sha1.New()

		if _, err = io.Copy(shaSum, chunkReader); err != nil {
			return err
		}

		expectedShaSum := fmt.Sprintf("%x", chunk.ShaHash)
		actualShaSum := fmt.Sprintf("%x", shaSum.Sum(nil))

		if expectedShaSum != actualShaSum {
			return errors.New("failed validation for " + chunkPath)
		}

		evca.Progress(chunk.FileSize)
	}

	evca.EndWithResult("valid")

	return nil
}

func egsDownloadChunks(appName string, ii *InstallInfo, originData *data.OriginData) error {

	edca := nod.NewProgress("downloading EGS chunks...")
	edca.Done()

	downloadsDir := camino.GetAbs(vangogh_integration.Downloads)

	if err := originHasFreeSpace(appName, downloadsDir, ii, originData); err != nil {
		return err
	}

	edca.Total(uint64(egsManifestSize(originData.Manifest)))

	cdnUrls, err := originData.GameManifest.Urls()
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	var cdnUrl *url.URL
	for _, cu := range cdnUrls {
		cdnUrl = cu
		break
	}

	if cdnUrl == nil {
		return errors.New("downloading EGS chunks requires CDN url")
	}

	absChunksDownloadsDir := data.AbsChunksDownloadDir(appName, ii.OperatingSystem)

	originalPath := strings.TrimSuffix(cdnUrl.Path, filepath.Base(cdnUrl.Path))
	cdnUrl.RawQuery = ""

	for _, chunk := range originData.Manifest.ChunkList.Chunks {

		chunkPath := chunk.Path(originData.Manifest.Metadata.FeatureLevel)
		cdnUrl.Path = path.Join(originalPath, chunkPath)

		if err = dc.Download(cdnUrl, ii.force, nil, absChunksDownloadsDir, chunkPath); err != nil {
			return err
		}

		edca.Progress(chunk.FileSize)
	}

	return nil
}

func egsGetExecTask(appName string, ii *InstallInfo, originData *data.OriginData, rdx redux.Writeable, et *execTask) (*execTask, error) {

	installedPath, err := originOsInstalledPath(appName, ii, rdx)
	if err != nil {
		return nil, err
	}

	absPrefixDir, err := data.AbsPrefixDir(appName, ii.Origin, rdx)
	if err != nil {
		return nil, err
	}

	launchDir, launchFile := filepath.Split(originData.Manifest.Metadata.LaunchExe)

	et.title = launchFile
	et.prefix = absPrefixDir
	et.exe = filepath.Join(installedPath, originData.Manifest.Metadata.LaunchExe)
	if originData.Manifest.Metadata.LaunchCommand != "" {
		et.args = append(et.args, originData.Manifest.Metadata.LaunchCommand)
	}
	et.workDir = filepath.Join(installedPath, launchDir)

	return et, nil
}
