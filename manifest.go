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
	tagManifest        = 1
	payloadManifest    = 2
	bagitPathSeparator = '/'
)

//Manifest represents a payload manifest file
type Manifest struct {
	algorithm string
	entries   map[string]ManifestEntry // key is encoded file path
	kind      int                      // tag or payload
}

type ManifestEntry struct {
	rawPath string
	size    int64       // filesize
	sum     string      // checksum
	notIn   []*Manifest // Manifests *missing* this entry
}

var manifestLineRE = regexp.MustCompile(`^(\S+)\s+(\S.*)$`)

var manifestFilenameRE = regexp.MustCompile(`(tag)?manifest-(\w+).txt$`)

// NewManifest returns an initialized manifest
func NewManifest(alg string) *Manifest {
	manifest := &Manifest{algorithm: alg}
	manifest.entries = make(map[string]ManifestEntry)
	return manifest
}

func ParseManifestEntries(reader io.Reader) (map[string]ManifestEntry, error) {
	entries := make(map[string]ManifestEntry)
	lineNum := 0
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lineNum++
		match := manifestLineRE.FindStringSubmatch(scanner.Text())
		if len(match) < 3 {
			msg := fmt.Sprintf("Syntax error at line: %d", lineNum)
			return nil, errors.New(msg)
		}
		_, exists := entries[match[2]]
		if exists {
			msg := fmt.Sprintf("Duplicate manifest entry at line: %d", lineNum)
			return nil, errors.New(msg)
		}
		cleanPath := filepath.Clean(decodePath(match[2]))
		if strings.HasPrefix(cleanPath, `..`) {
			msg := fmt.Sprintf("Out of scope path at line: %d", lineNum)
			return nil, errors.New(msg)
		}
		sum := strings.Trim(match[1], ` `)
		entries[encodePath(cleanPath)] = ManifestEntry{rawPath: cleanPath, sum: sum}
	}
	return entries, nil
}

// ReadManifest reads and parses a manifest file
func ReadManifest(p string) (*Manifest, error) {
	file, err := os.Open(p)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	manifest, err := NewManifestFromFilename(filepath.Base(p))
	if err != nil {
		return nil, err
	}
	manifest.entries, err = ParseManifestEntries(file)
	return manifest, err
}

// NewManifestFromFilename returns checksum algorithm from manifest's filename
func NewManifestFromFilename(filename string) (*Manifest, error) {
	manifest := &Manifest{}
	msg := fmt.Sprintf("Manifest filename not correctly formed: %s", filename)
	match := manifestFilenameRE.FindStringSubmatch(filename)
	if len(match) < 3 {
		return nil, errors.New(msg)
	}
	// Checksum algorithm
	alg, err := NormalizeAlgName(match[2])
	if err != nil {
		return nil, err
	}
	manifest.algorithm = alg
	// Manifest type
	if match[1] == `tag` {
		manifest.kind = tagManifest
	} else if match[1] == `` {
		manifest.kind = payloadManifest
	} else {
		return nil, errors.New(msg)
	}
	return manifest, nil
}

func ReadAllManifests(dir string) ([]*Manifest, error) {
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
		m, err = ReadManifest(f)
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
