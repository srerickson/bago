package bago

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/srerickson/bago/checksum"
)

const (
	payloadManifest = 0 // default
	tagManifest     = 1
)

//Manifest represents a payload manifest file
type Manifest struct {
	algorithm string
	entries   map[NormPath]ManifestEntry // key is unicode normalized
	kind      int                        // tag or payload
}

type ManifestEntry struct {
	path string // raw file system path
	sum  []byte
}

// Append adds a new entry to the manifest. It returns an error if the
// entry already exists. The path is encoded.
func (man *Manifest) Append(path EncPath, sum []byte) error {
	if man.entries == nil {
		man.entries = map[NormPath]ManifestEntry{}
	}
	var normPath = path.Norm()
	if _, exists := man.entries[normPath]; exists {
		return fmt.Errorf("duplicate entry")
	}
	man.entries[normPath] = ManifestEntry{path: path.Decode(), sum: sum}
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
		cleanEncPath := filepath.ToSlash(filepath.Clean(match[2]))
		if strings.HasPrefix(cleanEncPath, `..`) {
			return fmt.Errorf("Out of scope path at line: %d", lineNum)
		}
		var sum []byte
		var err error
		if sum, err = hex.DecodeString(strings.Trim(match[1], ` `)); err != nil {
			return fmt.Errorf("line %d: %s", lineNum, err.Error())
		}
		if err = man.Append(EncPath(cleanEncPath), sum); err != nil {
			return fmt.Errorf("line %d: %s", lineNum, err.Error())
		}
	}
	if lineNum == 0 {
		return errors.New("empty manifest")
	}
	return nil
}

func (man *Manifest) Write(writer io.Writer) error {
	for _, e := range man.entries {
		sum := hex.EncodeToString(e.sum)
		path := EncodePath(e.path)
		if _, err := fmt.Fprintf(writer, "%s %s\n", sum, path); err != nil {
			return err
		}
	}
	return nil
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
	alg, err := checksum.NormalizeAlgName(match[2])
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
