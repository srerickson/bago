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

// Append adds a new entry to the manifest. It returns an error if the
// entry already exists. The path is encoded.
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

func (man *Manifest) parse(reader io.Reader) error {
	manifestLineRE := regexp.MustCompile(`^(\S+)\s+(\S.*)$`)
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

func (man *Manifest) Write(writer io.Writer) {
	for k, v := range man.entries {
		fmt.Println(k)
		fmt.Fprintf(writer, "%s %s\n", k, v.sum)
	}
}

// Filename returns filename for the manifest
func (man *Manifest) Filename() string {
	if man.kind == tagManifest {
		return fmt.Sprintf("tagmanifest-%s.txt", man.algorithm)
	}
	return fmt.Sprintf("manifest-%s.txt", man.algorithm)
}

// NewManifestFromFilename returns new manifest based on filenme
func newManifestFromFilename(filename string) (*Manifest, error) {
	manifestFilenameRE := regexp.MustCompile(`^(tag)?manifest-(\w+).txt$`)
	match := manifestFilenameRE.FindStringSubmatch(filename)
	if len(match) < 3 {
		return nil, fmt.Errorf("Badly formed manifest filename: %s", filename)
	}
	alg, err := NormalizeAlgName(match[2])
	if err != nil {
		return nil, err
	}
	var kind int
	if match[1] == `tag` {
		kind = tagManifest
	} else if match[1] == `` {
		kind = payloadManifest
	} else {
		return nil, fmt.Errorf("Badly formed manifest filename: %s", filename)
	}
	return &Manifest{algorithm: alg, kind: kind}, nil
}
