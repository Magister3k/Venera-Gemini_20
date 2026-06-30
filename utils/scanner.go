package utils

import (
	"os"
	"path/filepath"
)

// ScanFiles возвращает список файлов в папке.
// Если scanSubfolders == true, сканирует рекурсивно.
func ScanFiles(root string, scanSubfolders bool) ([]string, error) {
	var files []string

	if !scanSubfolders {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, filepath.Join(root, e.Name()))
			}
		}
		return files, nil
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}
