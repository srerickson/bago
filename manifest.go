package bago

import (
	"bufio"
	"fmt"
	"io"
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
	entries   map[string]*ManifestEntry // key is encoded file path
	kind      int                       // tag or payload
}

type ManifestEntry struct {
	rawPath string
	sum     string // checksum
}

var manifestLineRE = regexp.MustCompile(`^(\S+)\s+(\S.*)$`)

func (man *Manifest) Append(path string, sum string) error {
	if man.entries == nil {
		man.entries = map[string]*ManifestEntry{}
	}
	encPath := encodePath(path)
	if _, exists := man.entries[encPath]; exists {
		return fmt.Errorf("duplicate entry")
	}
	man.entries[encPath] = &ManifestEntry{rawPath: path, sum: sum}
	return nil
}

// NewManifest returns an initialized manifest
func NewManifest(alg string) *Manifest {
	manifest := &Manifest{algorithm: alg}
	return manifest
}

func (man *Manifest) parse(reader io.Reader) error {
	lineNum := 0
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lineNum++
		match := manifestLineRE.FindStringSubmatch(scanner.Text())
		if len(match) < 3 {
			return fmt.Errorf("Syntax error at line: %d", lineNum)
		}
		cleanPath := filepath.Clean(decodePath(match[2]))
		if strings.HasPrefix(cleanPath, `..`) {
			return fmt.Errorf("Out of scope path at line: %d", lineNum)
		}
		sum := strings.Trim(match[1], ` `)
		err := man.Append(cleanPath, sum)
		if err != nil {
			return fmt.Errorf("line %d: %s", lineNum, err.Error())
		}
	}
	return nil
}

// ReadManifest reads and parses a manifest file

// NewManifestFromFilename returns checksum algorithm from manifest's filename
func newManifestFromFilename(filename string) (*Manifest, error) {
	manifestFilenameRE := regexp.MustCompile(`^(tag)?manifest-(\w+).txt$`)
	match := manifestFilenameRE.FindStringSubmatch(filename)
	if len(match) < 3 {
		return nil, fmt.Errorf("Badly formed manifest filename: %s", filename)
	}
	// Checksum algorithm
	alg, err := NormalizeAlgName(match[2])
	if err != nil {
		return nil, err
	}
	manifest := &Manifest{algorithm: alg}
	// Manifest type
	if match[1] == `tag` {
		manifest.kind = tagManifest
	} else if match[1] == `` {
		manifest.kind = payloadManifest
	} else {
		return nil, fmt.Errorf("Badly formed manifest filename: %s", filename)
	}
	return manifest, nil
}
