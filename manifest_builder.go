package bago

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ManifestBuilder scans files, computes checksums, and return manifests
type ManifestBuilder struct {
	Path    string
	Workers int // number of go routines for checksums
	Alg     string
	// includeHidden
}

type fileinfo struct {
	path string
	info os.FileInfo
	err  error
}

type checksum struct {
	path string
	hash string
	err  error
}

// Build builds manifest for given path, performing checksum
func (b *ManifestBuilder) Build() (*Manifest, error) {
	var manifest = NewManifest(b.Alg)
	var wg sync.WaitGroup

	// channels
	filenames := fileWalker(b.Path)
	checksums := make(chan checksum)

	//checksum workers
	for i := 0; i < b.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range filenames {
				if f.err != nil {
					checksums <- checksum{f.path, ``, f.err}
					break
				}
				sum, err := Checksum(f.path, b.Alg)
				if err != nil {
					checksums <- checksum{f.path, ``, err}
					break
				}
				checksums <- checksum{f.path, sum, nil}
			}
		}()
	}
	// Channel Closers
	go func() {
		wg.Wait()
		close(checksums)
	}()
	for sum := range checksums {
		relPath, _ := filepath.Rel(b.Path, sum.path)
		manifest.entries[relPath] = sum.hash
		fmt.Printf("%s %s\n", relPath, sum.hash)
	}
	return manifest, nil
}

// helper function that streams filenames to the retuned channel
func fileWalker(path string) chan fileinfo {
	files := make(chan fileinfo)
	go func(path string) {
		// stream filenames by walking filepath
    fmt.Println(path)
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if !info.Mode().IsDir() {
				files <- fileinfo{path: p, info: info, err: err}
			}
			return err // should be nil. If not, walk stops
		})
		close(files)
	}(path)
	return files
}
