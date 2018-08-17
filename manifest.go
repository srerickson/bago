package bago

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

//Manifest represents a payload manifest file
type Manifest struct {
	algorithm string
	entries   map[string]string // map: path -> checksum
}

var manifestLineRE = regexp.MustCompile(`^(\S+)\s+(\S.*\S)\s*$`)

var manifestFilenameRE = regexp.MustCompile(`.*manifest-(\w+).txt$`)

// NewManifest returns an initialized manifest
func NewManifest(alg string) *Manifest {
	manifest := &Manifest{algorithm: alg}
	manifest.entries = make(map[string]string)
	return manifest
}

//ReadManifest reads and parses a manifest file
func ParseManifest(path string) (*Manifest, []error) {
	errs := []error{}
	file, err := os.Open(path)
	if err != nil {
		return nil, append(errs, err)
	}
	defer file.Close()
	alg, err := ParseManifestFilename(path)
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

// ManifestAglorithm returns checksum algorithm from manifest's filename
func ParseManifestFilename(filename string) (string, error) {
	match := manifestFilenameRE.FindStringSubmatch(filename)
	if len(match) == 0 {
		return "", errors.New("Could not determine manifest's checksum algorithm")
	}
	alg := strings.ToLower(match[1])
	for _, a := range AvailableAlgs {
		if a == alg {
			return alg, nil
		}
	}
	msg := fmt.Sprintf("%s is not a recognized checksum algorithm", alg)
	return alg, errors.New(msg)
}
