package backend

import (
	"io"
	"os"
	"path/filepath"
)

type Backend interface {
	Stat(string) (os.FileInfo, error) // should throw error for directories
	Open(string) (io.ReadCloser, error)
	Create(string) (io.WriteCloser, error)
	Walk(root string, walkFn filepath.WalkFunc) error
}
