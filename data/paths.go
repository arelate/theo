package data

import (
	"github.com/boggydigital/pathways"
	"os"
	"path/filepath"
)

const theoDirname = "theo"

func InitRootDir() (string, error) {
	ucd, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	rootDir := filepath.Join(ucd, theoDirname)
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		if err := os.MkdirAll(rootDir, 0755); err != nil {
			return "", err
		}
	}

	for _, ad := range AllAbsDirs {
		absDir := filepath.Join(rootDir, string(ad))
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			if err := os.MkdirAll(absDir, 0755); err != nil {
				return "", err
			}
		}
	}

	for rd, ad := range RelToAbsDirs {
		absRelDir := filepath.Join(rootDir, string(ad), string(rd))
		if _, err := os.Stat(absRelDir); os.IsNotExist(err) {
			if err := os.MkdirAll(absRelDir, 0755); err != nil {
				return "", err
			}
		}
	}

	return filepath.Join(ucd, theoDirname), nil
}

const (
	Backups   pathways.AbsDir = "backups"
	Metadata  pathways.AbsDir = "metadata"
	Downloads pathways.AbsDir = "downloads"
	Extracts  pathways.AbsDir = "extracts"
	Cellars   pathways.AbsDir = "cellars"
)

const (
	Redux             pathways.RelDir = "_redux"
	DownloadsMetadata pathways.RelDir = "downloads-metadata"
	InstalledMetadata pathways.RelDir = "installed-metadata"
	GitHubReleases    pathways.RelDir = "github-releases"
	Releases          pathways.RelDir = "rel"
	Binaries          pathways.RelDir = "bin"
	Prefixes          pathways.RelDir = "pfx"
)

var RelToAbsDirs = map[pathways.RelDir]pathways.AbsDir{
	Redux:             Metadata,
	DownloadsMetadata: Metadata,
	InstalledMetadata: Metadata,
	GitHubReleases:    Metadata,
	Releases:          Cellars,
	Binaries:          Cellars,
	Prefixes:          Cellars,
}

var AllAbsDirs = []pathways.AbsDir{
	Backups,
	Metadata,
	Downloads,
	Extracts,
	Cellars,
}
