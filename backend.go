package bago

import "io"

type FileInfo struct {
	Path string
	Size int64
}

type Closable interface {
	Close() error
}

type Backend interface {
	Stat(string) (FileInfo, error)
	Open(string) (io.Reader, error)
	Close(*Closable) error
	Walk(string, func(string, FileInfo, error) error) error
	OpenBag(string) (*Bag, error)
	WriteBag(*Bag) (string, error)
	Checksum(string) (string, error)
}
