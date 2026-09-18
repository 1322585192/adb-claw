//go:build !unix

package pump

import (
	"os"
	"path/filepath"
)

func lockFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
}

func unlockFile(f *os.File) {
	if f != nil {
		_ = f.Close()
	}
}
