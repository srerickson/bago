package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/srerickson/bago"
)

// "github.com/srerickson/bago"

// type Backend interface {
// 	Stat(string) (FileInfo, error)
// 	Open(string) (io.Reader, error)
// 	Walk(string, func(string, FileInfo, error) error) error
// 	OpenBag(string) (*Bag, error)
// 	WriteBag(*Bag) (string, error)
// 	Checksum(string) (string, error)
// }

type FSBackend struct {
	path string
}

func (be *FSBackend) Stat(path string) (bago.FileInfo, error) {
	fi := bago.FileInfo{}
	info, err := os.Stat(filepath.Join(be.path, path))
	if err != nil {
		return fi, err
	}
	if info.IsDir() {
		return fi, fmt.Errorf("%s is a directory", path)
	}
	fi.Path, fi.Size = path, info.Size()
	return fi, nil
}

func (be *FSBackend) Open(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(be.path, path))
}

func (be *FSBackend) Close(file io.Closer) error {
	return file.Close()
}

func (be *FSBackend) Walk(p string, f func(string, int64, error) error) error {
	wrapF := func(path string, fi os.FileInfo, err error) error {
		if err != nil || (fi != nil && fi.IsDir()) {
			return err
		}
		relPath, _ := filepath.Rel(be.path, path)
		return f(relPath, fi.Size(), err)
	}
	return filepath.Walk(filepath.Join(be.path, p), wrapF)
}

func (be *FSBackend) Checksum(path string) (string, error) {
	return ``, nil
}

func (be *FSBackend) AllManifests() []string {
	manFiles, err := filepath.Glob(filepath.Join(be.path, "*manifest-*.txt"))
	for i := range manFiles {
		manFiles[i], _ = filepath.Rel(be.path, manFiles[i])
	}
	if err != nil {
		return nil
	}
	return manFiles
}

func OpenBag(path string) (*bago.Bag, error) {
	backend := &FSBackend{path: path}
	bag := &bago.Bag{Backend: backend}
	return bag, bag.Hydrate()
}

func WriteBag(bag *bago.Bag) (string, error) {
	return ``, nil
}
