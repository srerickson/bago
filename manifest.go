package bago

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

var manifestLineRE = regexp.MustCompile(`^(\S+)\s+(\S.*\S)\s*$`)

//Manifest represents a payload manifest file
type Manifest struct {
	algorithm string
	entries   map[string]string // map: path -> checksum
}

type fileWalk struct {
	path string
	info os.FileInfo
	err  error
}

type checksum struct {
	path string
	hash string
	err  error
}

// NewManifest returns an initialized manifest
func NewManifest(alg string) *Manifest {
	manifest := &Manifest{algorithm: alg}
	manifest.entries = make(map[string]string)
	return manifest
}

//ReadManifest reads and parses a manifest file
func ReadManifest(path string) (*Manifest, []error) {
	errs := []error{}
	file, err := os.Open(path)
	if err != nil {
		return nil, append(errs, err)
	}
	defer file.Close()
	alg, err := ManifestAglorithm(path)
	if err != nil {
		return nil, append(errs, err)
	}
	manifest := NewManifest(alg)
	lineNum := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		match := manifestLineRE.FindStringSubmatch(scanner.Text())
		if len(match) < 3 {
			msg := fmt.Sprintf("error on line %d of %s", lineNum, path)
			errs = append(errs, errors.New(msg))
		} else {
			manifest.entries[match[2]] = match[1]
		}
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return manifest, nil
}

func fileWalker(path string) chan fileWalk {
	files := make(chan fileWalk)
	go func() {
		// stream filenames by walking filepath
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if !info.Mode().IsDir() {
				files <- fileWalk{path: p, info: info, err: err}
			}
			return err // should be nil. If not, walk stops
		})
		close(files)
	}()
	return files
}

// GenerateManifest builds manifest for given path, performing checksum
func GenerateManifest(path string, workers int, alg string) (*Manifest, error) {
	var manifest = NewManifest(alg)
	var wg sync.WaitGroup

	// channels
	filenames := fileWalker(path)
	checksums := make(chan checksum)

	//checksum workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range filenames {
				if f.err != nil {
					checksums <- checksum{f.path, ``, f.err}
					break
				}
				sum, err := Checksum(f.path, alg)
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
		relPath, _ := filepath.Rel(path, sum.path)
		manifest.entries[relPath] = sum.hash
		fmt.Printf("%s %s\n", relPath, sum.hash)
	}

	return manifest, nil
}
