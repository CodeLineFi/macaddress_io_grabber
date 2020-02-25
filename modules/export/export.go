package export

import (
	"os"
	"path/filepath"
)

func createFile(filename string) (*os.File, error) {
	dir, _ := filepath.Split(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.Create(filename)
}
