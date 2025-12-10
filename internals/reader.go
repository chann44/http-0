package internals

import (
	"os"
	"path/filepath"
	"strings"
)

func ReadFileToBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func GetFiles(cwd string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".yml" || ext == ".yaml" {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
