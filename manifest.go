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

// GenerateManifest builds manifest for given path, performing checksum
func GenerateManifest(path string, workers int, alg string) (*Manifest, error) {
	var manifest = NewManifest(alg)
	var wg, wg2 sync.WaitGroup

	// channels
	filenames := make(chan string)
	checksums := make(chan [2]string)
	errs := make(chan error)

	// stream filenames by walking filepath
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("error accessing a path %q: %v\n", path, err)
			errs <- err
			return nil
		}
		// skip directories
		if info.Mode().IsDir() {
			return nil
		}
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			filenames <- p
		}(p)
		return nil
	})

	//checksum workers
	for i := 0; i < workers; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			for f := range filenames {
				sum, err := Checksum(f, alg)
				if err != nil {
					errs <- err
				} else {
					checksums <- [2]string{f, sum}
				}
			}
		}()
	}

	// Channel Closers
	go func() {
		wg.Wait()
		close(filenames)
	}()
	go func() {
		wg2.Wait()
		close(checksums)
	}()

	for sum := range checksums {
		relPath, _ := filepath.Rel(path, sum[0])
		manifest.entries[relPath] = sum[1]
		fmt.Printf("%s %s\n", relPath, sum[1])
	}

	return manifest, nil
}
