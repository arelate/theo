package cli

import (
	"io/fs"
	"path/filepath"
	"slices"
)

const (
	dsStoreFilename = ".DS_Store"
)

var ignoredFilenames = []string{
	dsStoreFilename,
}

func relWalkDir(absPath string) ([]string, error) {

	files := make([]string, 0)

	if err := filepath.Walk(absPath, func(path string, info fs.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if slices.Contains(ignoredFilenames, info.Name()) {
			return nil
		}

		var relPath string
		relPath, err = filepath.Rel(absPath, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)

		return nil

	}); err != nil {
		return nil, err
	}

	return files, nil
}
