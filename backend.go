package bago

import "io"

type FileInfo struct {
	Path string
	Size int64
}

type Backend interface {
	Stat(string) (FileInfo, error)
	Open(string) (io.ReadCloser, error)
	AllManifests() []string
	Close(io.Closer) error
	Walk(string, func(string, int64, error) error) error
	// OpenBag(string) (*Bag, error)
	// WriteBag(*Bag) (string, error)
	Checksum(string) (string, error)
}
