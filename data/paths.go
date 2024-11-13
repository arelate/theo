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

	return filepath.Join(ucd, theoDirname), nil
}

const (
	Backups   pathways.AbsDir = "backups"
	Metadata  pathways.AbsDir = "metadata"
	Downloads pathways.AbsDir = "downloads"
)

var AllAbsDirs = []pathways.AbsDir{
	Backups,
	Metadata,
	Downloads,
}
