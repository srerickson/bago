package backend

import (
	"io"
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
}
