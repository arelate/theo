package cli

import (
	"bufio"
	_ "embed"
	"github.com/arelate/southern_light/vangogh_integration"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	bundleNamePfx    = "gog_bundle_name="
	installerTypePfx = "gog_installer_type="

	relPostInstallScriptPath = "package.pkg/Scripts/postinstall"
)

//go:embed "known_postinstall.txt"
var knownPostInstall string

type PostInstallScript struct {
	path           string
	bundleName     string
	installerType  string
	customCommands []string
}

func (pis *PostInstallScript) BundleName() string {
	return pis.bundleName
}

func (pis *PostInstallScript) InstallerType() string {
	return pis.installerType
}

func (pis *PostInstallScript) CustomCommands() []string {
	return pis.customCommands
}

func ParsePostInstallScript(path string) (*PostInstallScript, error) {

	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	scriptFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer scriptFile.Close()

	pis := &PostInstallScript{}
	knownPostInstallLines := strings.Split(knownPostInstall, "\n")

	scriptScanner := bufio.NewScanner(scriptFile)
	for scriptScanner.Scan() {

		if err := scriptScanner.Err(); err != nil {
			return nil, err
		}

		line := scriptScanner.Text()
		if line == "" {
			continue
		}

		if after, ok := strings.CutPrefix(line, bundleNamePfx); ok {
			pis.bundleName = strings.Trim(after, "\"")
			continue
		}
		if after, ok := strings.CutPrefix(line, installerTypePfx); ok {
			pis.installerType = strings.Trim(after, "\"")
			continue
		}
		if !slices.Contains(knownPostInstallLines, line) {
			pis.customCommands = append(pis.customCommands, line)
		}
	}

	return pis, nil
}

func PostInstallScriptPath(productExtractsDir string, link *vangogh_integration.ProductDownloadLink) string {
	localFilenameExtractsDir := filepath.Join(productExtractsDir, link.LocalFilename)
	return filepath.Join(localFilenameExtractsDir, relPostInstallScriptPath)
}
