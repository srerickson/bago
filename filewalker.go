package bago

import (
	"os"
	"path/filepath"
)

// FileInfo is a structure for streaming file information
type fileInfo struct {
	path string
	info os.FileInfo
	err  error
}

// FileWalker is a helper function that streams filenames to the retuned channel
func fileWalker(path string) chan fileInfo {
	files := make(chan fileInfo)
	go func(path string) {
		// stream filenames by walking filepath
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err == nil && info != nil && info.IsDir() {
				return nil // skip directories
			}
			files <- fileInfo{path: p, info: info, err: err}
			return err // should be nil. If not, filepath.Walk stops
		})
		close(files)
	}(path)
	return files
}
