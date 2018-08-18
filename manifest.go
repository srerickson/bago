package bago

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	tagManifest     = 1
	payloadManifest = 2
)

//Manifest represents a payload manifest file
type Manifest struct {
	algorithm string
	entries   map[string]string // map: p -> checksum
	kind      int               // tagManifest || payloadManifest
}

var manifestLineRE = regexp.MustCompile(`^(\S+)\s+(\S.*\S)\s*$`)

var manifestFilenameRE = regexp.MustCompile(`.*manifest-(\w+).txt$`)

var payloadManifestFilenameRE = regexp.MustCompile(`^manifest-(\w+).txt$`)

var tagManifestFilenameRE = regexp.MustCompile(`^tagmanifest-(\w+).txt$`)

// NewManifest returns an initialized manifest
func NewManifest(alg string) *Manifest {
	manifest := &Manifest{algorithm: alg}
	manifest.entries = make(map[string]string)
	return manifest
}

// ParseManifest reads and parses a manifest file
func ParseManifest(p string) (*Manifest, error) {
	file, err := os.Open(p)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	manifest, err := NewManifestFromFilename(path.Base(p))
	manifest.entries = make(map[string]string)
	if err != nil {
		return nil, err
	}
	lineNum := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		match := manifestLineRE.FindStringSubmatch(scanner.Text())
		if len(match) < 3 {
			msg := fmt.Sprintf("error on line %d of %s", lineNum, p)
			return manifest, errors.New(msg)
		}
		manifest.entries[match[2]] = match[1]
	}
	return manifest, nil
}

// NewManifestFromFilename returns checksum algorithm from manifest's filename
func NewManifestFromFilename(filename string) (*Manifest, error) {
	manifest := &Manifest{}
	// determine algorithm
	match := manifestFilenameRE.FindStringSubmatch(filename)
	if len(match) < 2 {
		return manifest, errors.New("Could not determine manifest algorithm")
	}
	alg := strings.ToLower(match[1])
	for _, a := range AvailableAlgs {
		if a == alg {
			manifest.algorithm = alg
			break
		}
	}
	if manifest.algorithm == `` {
		msg := fmt.Sprintf("%s is not a recognized checksum algorithm", alg)
		return manifest, errors.New(msg)
	}
	// determine manifest type
	if payloadManifestFilenameRE.MatchString(filename) {
		manifest.kind = payloadManifest
	} else if tagManifestFilenameRE.MatchString(filename) {
		manifest.kind = tagManifest
	} else {
		msg := fmt.Sprintf("Could not determine manifest type")
		return manifest, errors.New(msg)
	}
	return manifest, nil
}

func ParseAllManifests(dir string) ([]*Manifest, error) {
	//parse manifest files
	mans := []*Manifest{}

	manFiles, err := filepath.Glob(filepath.Join(dir, "*manifest-*.txt"))
	if err != nil {
		return mans, err
	}
	if len(manFiles) == 0 {
		return mans, errors.New(`No manifest files found`)
	}

	for _, f := range manFiles {
		var m *Manifest
		m, err = ParseManifest(f)
		if err != nil {
			return mans, err
		}
		mans = append(mans, m)
	}
	return mans, nil
}
