package backend

import (
	"io"
	"os"
	"path/filepath"
)

// FS Implements Backend for the filesystem
type FS struct {
	Path string
}

func (be *FS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(filepath.Join(be.Path, path))
}

func (be *FS) Open(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(be.Path, path))
}

func (be *FS) Create(path string) (io.WriteCloser, error) {
	return os.Create(filepath.Join(be.Path, path))
}

func (be *FS) Walk(p string, f filepath.WalkFunc) error {
	wrapF := func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.Mode().IsRegular() {
			return err
		}
		relPath, _ := filepath.Rel(be.Path, path)
		return f(relPath, fi, err)
	}
	return filepath.Walk(filepath.Join(be.Path, p), wrapF)
}
