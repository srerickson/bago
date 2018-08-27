package bago

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
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

// NewManifest returns an initialized manifest
func NewManifest(alg string) *Manifest {
	manifest := &Manifest{algorithm: alg}
	return manifest
}

func (man *Manifest) parseEntries(reader io.Reader) error {
	man.entries = make(map[string]*ManifestEntry)
	lineNum := 0
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lineNum++
		match := manifestLineRE.FindStringSubmatch(scanner.Text())
		if len(match) < 3 {
			return fmt.Errorf("Syntax error at line: %d", lineNum)
		}
		_, exists := man.entries[match[2]]
		if exists {
			return fmt.Errorf("Duplicate manifest entry at line: %d", lineNum)
		}
		cleanPath := filepath.Clean(decodePath(match[2]))
		if strings.HasPrefix(cleanPath, `..`) {
			return fmt.Errorf("Out of scope path at line: %d", lineNum)
		}
		sum := strings.Trim(match[1], ` `)

		man.entries[encodePath(cleanPath)] = &ManifestEntry{
			rawPath: cleanPath, sum: sum}
	}
	return nil
}

// ReadManifest reads and parses a manifest file
func ReadManifest(p string, enc string) (*Manifest, error) {
	file, err := os.Open(p)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	manifest, err := NewManifestFromFilename(filepath.Base(p))
	if err != nil {
		return nil, err
	}
	decodeReader, err := newReader(file, enc)
	if err != nil {
		return nil, err
	}
	return manifest, manifest.parseEntries(decodeReader)
}

// NewManifestFromFilename returns checksum algorithm from manifest's filename
func NewManifestFromFilename(filename string) (*Manifest, error) {
	manifestFilenameRE := regexp.MustCompile(`(tag)?manifest-(\w+).txt$`)
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

func ReadAllManifests(dir string, enc string) ([]*Manifest, error) {
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
		m, err = ReadManifest(f, enc)
		if err != nil {
			return mans, err
		}
		mans = append(mans, m)
	}
	return mans, nil
}

func encodePath(s string) string {
	s = norm.NFC.String(s) // Not sure this should be here
	s = strings.Replace(s, `%`, `%25`, -1)
	s = strings.Replace(s, "\r", `%0D`, -1)
	s = strings.Replace(s, "\n", `%0A`, -1)
	s = filepath.ToSlash(s)
	return s
}

func decodePath(s string) string {
	lf := regexp.MustCompile(`(%0[Aa])`)
	cr := regexp.MustCompile(`(%0[Dd])`)
	s = filepath.FromSlash(s)
	s = lf.ReplaceAllString(s, "\n")
	s = cr.ReplaceAllString(s, "\r")
	s = strings.Replace(s, `%25`, `%`, -1)
	return s
}
