package data

import (
	"bytes"
	_ "embed"
	"errors"
	"github.com/arelate/vangogh_local_data"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/wits"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	wineSourcesFilename = "wine.txt"
	dxVkSourcesFilename = "dxvk.txt"
)

var (
	//go:embed "default_sources/wine.txt"
	defaultWineSources []byte
	//go:embed "default_sources/dxvk.txt"
	defaultDxVkSources []byte
)

type GitHubSource struct {
	OS           vangogh_local_data.OperatingSystem
	Owner        string
	Repo         string
	Description  string
	AssetInclude []string
	AssetExclude []string
	Default      bool
}

type WineGitHubSource struct {
	*GitHubSource
	BinaryPath string
}

func (ghs *GitHubSource) String() string {
	return path.Join(ghs.Owner, ghs.Repo)
}

func (ghs *GitHubSource) CurrentOsMatches(owner, repo string) bool {
	if ghs.OS == CurrentOS() &&
		ghs.Repo == repo {
		if (owner != "" && ghs.Owner == owner) || owner == "" {
			return true
		}
	}
	return false
}

func parseGitHubSource(u *url.URL, pkv wits.KeyValue) (*GitHubSource, error) {

	owner, repo := path.Split(u.Path)
	owner = strings.Trim(owner, "/")

	ghs := &GitHubSource{
		Owner: owner,
		Repo:  repo,
	}

	for key, value := range pkv {
		switch key {
		case "os":
			if os := vangogh_local_data.ParseOperatingSystem(value); os != vangogh_local_data.AnyOperatingSystem {
				ghs.OS = os
			} else {
				return nil, errors.New("WINE source specifies unknown operating system")
			}
		case "desc":
			ghs.Description = value
		case "assets-include":
			ghs.AssetInclude = strings.Split(value, ";")
		case "assets-exclude":
			ghs.AssetExclude = strings.Split(value, ";")
		}
	}

	return ghs, nil
}

func parseWineSource(u *url.URL, pkv wits.KeyValue) (*WineGitHubSource, error) {

	ghs, err := parseGitHubSource(u, pkv)
	if err != nil {
		return nil, err
	}

	wineSource := &WineGitHubSource{
		GitHubSource: ghs,
	}

	for key, value := range pkv {
		switch key {
		case "default":
			wineSource.Default = value == "true"
		case "bin-path":
			wineSource.BinaryPath = value
		}
	}

	return wineSource, nil
}

func loadGitHubSectionKeyValue(relSourcePath string) (map[*url.URL]wits.KeyValue, error) {

	githubSourcesDir, err := pathways.GetAbsDir(GitHubSources)
	if err != nil {
		return nil, err
	}

	absSourcesPath := filepath.Join(githubSourcesDir, relSourcePath)

	sourcesFile, err := os.Open(absSourcesPath)
	if err != nil {
		return nil, err
	}

	skvSources, err := wits.ReadSectionKeyValue(sourcesFile)
	if err != nil {
		return nil, err
	}

	urlKv := make(map[*url.URL]wits.KeyValue)

	for urlStr, kv := range skvSources {
		sourceUrl, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}

		urlKv[sourceUrl] = kv
	}

	return urlKv, nil
}

func LoadWineSources() ([]*WineGitHubSource, error) {

	urlKv, err := loadGitHubSectionKeyValue(wineSourcesFilename)
	if err != nil {
		return nil, err
	}

	wineSources := make([]*WineGitHubSource, 0)

	for sourceUrl, kv := range urlKv {
		wineSource, err := parseWineSource(sourceUrl, kv)
		if err != nil {
			return nil, err
		}
		wineSources = append(wineSources, wineSource)
	}

	return wineSources, nil
}

func LoadDxVkSources() ([]*GitHubSource, error) {
	urlKv, err := loadGitHubSectionKeyValue(dxVkSourcesFilename)
	if err != nil {
		return nil, err
	}

	dxVkSources := make([]*GitHubSource, 0)

	for sourceUrl, kv := range urlKv {
		dxVkSource, err := parseGitHubSource(sourceUrl, kv)
		if err != nil {
			return nil, err
		}
		dxVkSources = append(dxVkSources, dxVkSource)
	}

	return dxVkSources, nil
}

func LoadGitHubSources() ([]*GitHubSource, error) {
	githubSources := make([]*GitHubSource, 0)

	wineSources, err := LoadWineSources()
	if err != nil {
		return nil, err
	}
	for _, ws := range wineSources {
		githubSources = append(githubSources, ws.GitHubSource)
	}

	dxVkSources, err := LoadDxVkSources()
	if err != nil {
		return nil, err
	}
	githubSources = append(githubSources, dxVkSources...)

	return githubSources, nil
}

func splitOwnerRepo(gitHubRepo string) (string, string) {
	owner, repo := "", ""
	if strings.Contains(gitHubRepo, "/") {
		owner, repo = path.Split(gitHubRepo)
		owner = strings.TrimSuffix(owner, "/")
	} else {
		repo = gitHubRepo
	}
	return owner, repo
}

func GetWineSource(wineRepo string) (*WineGitHubSource, error) {

	owner, repo := splitOwnerRepo(wineRepo)
	if repo == "" {
		return getDefaultWineSource()
	}

	wineSources, err := LoadWineSources()
	if err != nil {
		return nil, err
	}

	for _, ws := range wineSources {
		if ws.CurrentOsMatches(owner, repo) {
			return ws, nil
		}
	}
	return nil, errors.New("WINE source not found")
}

func getDefaultWineSource() (*WineGitHubSource, error) {

	wineSources, err := LoadWineSources()
	if err != nil {
		return nil, err
	}

	for _, ws := range wineSources {
		if ws.OS == CurrentOS() &&
			ws.Default {
			return ws, nil
		}
	}

	return nil, errors.New("cannot determine default WINE source for " + CurrentOS().String())
}

func getDefaultDxVkSource() (*GitHubSource, error) {
	dxVkSources, err := LoadDxVkSources()
	if err != nil {
		return nil, err
	}

	for _, ds := range dxVkSources {
		if ds.OS == CurrentOS() &&
			ds.Default {
			return ds, nil
		}
	}

	return nil, errors.New("cannot determine default DXVK source for " + CurrentOS().String())
}

func GetDxVkSource(dxVkRepo string) (*GitHubSource, error) {

	owner, repo := splitOwnerRepo(dxVkRepo)
	if repo == "" {
		return getDefaultDxVkSource()
	}

	githubSources, err := LoadDxVkSources()
	if err != nil {
		return nil, err
	}

	for _, gs := range githubSources {
		if gs.CurrentOsMatches(owner, repo) {
			return gs, nil
		}
	}
	return nil, errors.New("DXVK source not found")
}

func InitGitHubSources() error {

	githubSourcesDir, err := pathways.GetAbsDir(GitHubSources)
	if err != nil {
		return err
	}

	absWineSourcesPath := filepath.Join(githubSourcesDir, wineSourcesFilename)
	if err := createIfNotExist(absWineSourcesPath, defaultWineSources); err != nil {
		return err
	}

	absDxVkSourcesPath := filepath.Join(githubSourcesDir, dxVkSourcesFilename)
	if err := createIfNotExist(absDxVkSourcesPath, defaultDxVkSources); err != nil {
		return err
	}

	return nil
}

func createIfNotExist(absPath string, defaultContent []byte) error {
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		wsFile, err := os.Create(absPath)
		if err != nil {
			return err
		}

		defer wsFile.Close()

		if _, err := io.Copy(wsFile, bytes.NewReader(defaultContent)); err != nil {
			return err
		}
	}
	return nil
}
