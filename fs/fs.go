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
	if !info.IsDir() {
		return fi, fmt.Errorf("%s is a directory", path)
	}
	fi.Path, fi.Size = path, info.Size()
	return fi, nil
}

func (be *FSBackend) Open(path string) (io.Reader, error) {
	return os.Open(filepath.Join(be.path, path))
}

func (be *FSBackend) Close(file bago.Closable) error {
	return file.Close()
}

func (be *FSBackend) Walk(p string, f func(string, bago.FileInfo, error) error) error {
	wrapF := func(path string, fi os.FileInfo, err error) error {
		if err != nil || (fi != nil && fi.IsDir()) {
			return err
		}
		relPath, _ := filepath.Rel(p, path)
		wrapFI := bago.FileInfo{Path: relPath, Size: fi.Size()}
		return f(relPath, wrapFI, err)
	}
	return filepath.Walk(p, wrapF)
}

// func (*FSBackend) OpenBag(string) (*Bag, error)
// func (*FSBackend) WriteBag(*Bag) (string, error)
// func (*FSBackend) Checksum(string) (string, error)
