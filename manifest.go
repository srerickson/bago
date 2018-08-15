package bago

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
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

//LoadManifest reads and parses a manifest file
func LoadManifest(path string) (*Manifest, []error) {
	errs := []error{}
	file, err := os.Open(path)
	if err != nil {
		return nil, append(errs, err)
	}
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
