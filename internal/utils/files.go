package utils

import (
	"io/fs"
	"path/filepath"
)

// FindCdtFiles recursively finds all .cdt files in the specified directory
func FindCdtFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file has .cdt extension
		if filepath.Ext(path) == ".cdt" {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
