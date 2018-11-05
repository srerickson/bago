package backend

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FS Implements Backend for the filesystem
type FS struct {
	Path string
}

func (be *FS) Stat(path string) (FileInfo, error) {
	fi := FileInfo{}
	info, err := os.Stat(filepath.Join(be.Path, path))
	if err != nil {
		return fi, err
	}
	if info.IsDir() {
		return fi, fmt.Errorf("%s is a directory", path)
	}
	fi.Path, fi.Size = path, info.Size()
	return fi, nil
}

func (be *FS) Open(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(be.Path, path))
}

func (be *FS) Create(path string) (io.WriteCloser, error) {
	return os.Create(filepath.Join(be.Path, path))
}

func (be *FS) Walk(p string, f func(string, int64, error) error) error {
	wrapF := func(path string, fi os.FileInfo, err error) error {
		if err != nil || (fi != nil && fi.IsDir()) {
			return err
		}
		relPath, _ := filepath.Rel(be.Path, path)
		return f(relPath, fi.Size(), err)
	}
	return filepath.Walk(filepath.Join(be.Path, p), wrapF)
}

func (be *FS) AllManifests() []string {
	manFiles, err := filepath.Glob(filepath.Join(be.Path, "*manifest-*.txt"))
	for i := range manFiles {
		manFiles[i], _ = filepath.Rel(be.Path, manFiles[i])
	}
	if err != nil {
		return nil
	}
	return manFiles
}
