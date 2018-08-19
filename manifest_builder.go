package bago

import (
	"fmt"
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

type checksum struct {
	path string
	hash string
	size int64
	err  error
}

// Build builds manifest for given path, performing checksum
func (b *ManifestBuilder) Build(files chan fileInfo) (*Manifest, error) {
	manifest := NewManifest(b.Alg)
	checksums := make(chan checksum)
	var wg sync.WaitGroup

	//checksum workers
	for i := 0; i < b.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range files {
				if f.err != nil {
					checksums <- checksum{f.path, ``, 0, f.err}
					break
				}
				var err error
				sum := ``
				size := f.info.Size()
				if b.Alg != "" {
					sum, err = Checksum(f.path, b.Alg)
				}
				checksums <- checksum{f.path, sum, size, err}
				if err != nil {
					break
				}
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
		manifest.entries[encodePath(relPath)] = ManifestEntry{rawPath: sum.path, sum: sum.hash}
		fmt.Printf("%s %s\n", relPath, sum.hash)
	}
	return manifest, nil
}

func (b *ManifestBuilder) BuildList(files []fileInfo) (*Manifest, error) {
	filechan := make(chan fileInfo)
	go func(files []fileInfo) {
		for _, f := range files {
			filechan <- f
		}
		close(filechan)
	}(files)
	return b.Build(filechan)
}
