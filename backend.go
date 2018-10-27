package bago

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FileInfo struct {
	Path string // path should be relative to bag
	Size int64
}

type Backend interface {
	Stat(string) (FileInfo, error) // should throw error for directories
	Open(string) (io.ReadCloser, error)
	Create(string) (io.WriteCloser, error)
	AllManifests() []string
	Walk(string, func(string, int64, error) error) error
	Checksum(path string, alg string) (string, error)
}

// FSBag Implements Backend for the filesystem
type FSBag struct {
	path string
}

func (be *FSBag) Stat(path string) (FileInfo, error) {
	fi := FileInfo{}
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

func (be *FSBag) Open(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(be.path, path))
}

func (be *FSBag) Create(path string) (io.WriteCloser, error) {
	return os.Create(filepath.Join(be.path, path))
}

func (be *FSBag) Walk(p string, f func(string, int64, error) error) error {
	wrapF := func(path string, fi os.FileInfo, err error) error {
		if err != nil || (fi != nil && fi.IsDir()) {
			return err
		}
		relPath, _ := filepath.Rel(be.path, path)
		return f(relPath, fi.Size(), err)
	}
	return filepath.Walk(filepath.Join(be.path, p), wrapF)
}

func (be *FSBag) AllManifests() []string {
	manFiles, err := filepath.Glob(filepath.Join(be.path, "*manifest-*.txt"))
	for i := range manFiles {
		manFiles[i], _ = filepath.Rel(be.path, manFiles[i])
	}
	if err != nil {
		return nil
	}
	return manFiles
}

func (be *FSBag) Checksum(path string, alg string) (string, error) {
	h, err := NewHash(alg)
	if err != nil {
		return "", err
	}
	file, err := be.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = io.Copy(h, file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
