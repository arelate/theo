package cli

import (
	"bufio"
	_ "embed"
	"golang.org/x/exp/slices"
	"os"
	"strings"
)

const (
	bundleNamePfx    = "gog_bundle_name="
	installerTypePfx = "gog_installer_type="
)

//go:embed "knownPostInstallPayload.txt"
var knownPostInstallPayload string

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
	knownPostInstallLines := strings.Split(knownPostInstallPayload, "\n")

	scriptScanner := bufio.NewScanner(scriptFile)
	for scriptScanner.Scan() {

		if err := scriptScanner.Err(); err != nil {
			return nil, err
		}

		line := scriptScanner.Text()
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, bundleNamePfx) {
			pis.bundleName = strings.Trim(strings.TrimPrefix(line, bundleNamePfx), "\"")
			continue
		}
		if strings.HasPrefix(line, installerTypePfx) {
			pis.installerType = strings.Trim(strings.TrimPrefix(line, installerTypePfx), "\"")
			continue
		}
		if !slices.Contains(knownPostInstallLines, line) {
			pis.customCommands = append(pis.customCommands, line)
		}
	}

	return pis, nil
}
