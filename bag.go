package bago

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	defaultVersion = `0.97` // BagIt version for new bags
	bagitTxt       = `bagit.txt`
	bagInfo        = `bag-info.txt`
	fetchTxt       = `fetch.txt`
	dataDir        = `data`
)

// Bag is a bagit repository
type Bag struct {
	Backend      Backend     // backend interface (usually FSBag)
	version      [2]int      // from bagit txt, major and minor ints
	encoding     string      // from bagit.txt
	payload      Payload     // contents of the data directory
	Info         TagFile     // contents of bag-info.txt
	manifests    []*Manifest // list of payload manifests
	tagManifests []*Manifest // list of tag file manifests
	fetch        fetch       // contents of fetch.txt
}

type Payload map[string]PayloadEntry

type PayloadEntry struct {
	rawPath string
	size    int64
}

// TODO bad function name
func (bag *Bag) Hydrate() error {
	if bag.Backend == nil {
		return errors.New("Cannot hydrate a bag with no Backend\n")
	}
	err := bag.readBagitTxt()
	if err != nil {
		return err
	}
	_ = bag.readBagInfo()
	err = bag.readFetchFile()
	if err != nil {
		return err
	}
	err = bag.readPayload()
	if err != nil {
		return err
	}
	err = bag.readAllManifests()
	if err != nil {
		return err
	}
	return nil
}

// IsComplete returns whether bag satisfies bag completeness conditions.
// See: https://tools.ietf.org/html/draft-kunze-bagit-16#section-3
func (b *Bag) IsComplete() (bool, error) {
	if b.encoding == `` || !b.versionOk() {
		return false, fmt.Errorf("Missing required fields in %s", bagitTxt)
	}
	if b.payload == nil {
		return false, fmt.Errorf("bag has no payload")
	}
	if len(b.manifests) == 0 {
		return false, fmt.Errorf("bag has no manifest")
	}
	missing := b.notInPayload()
	if len(missing) > 0 {
		msg := "Manifest files missing from payload:"
		return false, fmt.Errorf("%s %s", msg, strings.Join(missing, `\n -`))
	}
	missing = b.notInManifests(0)
	if len(missing) > 0 {
		msg := "Payload files missing from manifest:"
		return false, fmt.Errorf("%s %s", msg, strings.Join(missing, `\n -`))
	}
	missing = b.missingTagFiles()
	if len(missing) > 0 {
		msg := "Tagfiles missing from tag manifests:"
		return false, fmt.Errorf("%s %s", msg, strings.Join(missing, `\n -`))
	}
	return true, nil
}

// IsValid returns whether the bag at path is valid
// A valid bag is complete and checksums listed in all manifests are correct.
func (b *Bag) IsValidConcurrent(workers int) (bool, error) {
	if _, err := b.IsComplete(); err != nil {
		return false, fmt.Errorf(`Bag is not complete: %s`, err.Error())
	}
	if err := b.ValidateManifests(workers); err != nil {
		return false, err
	}
	return true, nil
}

func (b *Bag) IsValid() (bool, error) {
	return b.IsValidConcurrent(1)
}

// missingTagFiles scans tag manifest entries and reports missing tag files
func (bag *Bag) missingTagFiles() []string {
	missing := []string{}
	for _, m := range bag.tagManifests {
		for _, tEntry := range m.entries {
			_, err := bag.Backend.Stat(tEntry.rawPath)
			if err != nil {
				missing = append(missing, err.Error())
			}
		}
	}
	return missing
}

// notInPayload scans manifests for files not present in the payload.
func (b *Bag) notInPayload() []string {
	missing := []string{}
	for _, m := range b.manifests {
		for mPath := range m.entries {
			if _, ok := b.payload[mPath]; !ok {
				missing = append(missing, mPath)
			}
		}
	}
	return missing
}

// notInManifests scans payload for files not accounted for in manifests
// thesh is the min number of manifests that a file can be missing from
func (b *Bag) notInManifests(thresh int) []string {
	counts := make(map[string]int)
	missing := []string{}
	for pPath := range b.payload {
		for _, man := range b.manifests {
			if _, ok := man.entries[pPath]; !ok {
				if counts[pPath]++; counts[pPath] > thresh {
					missing = append(missing, pPath)
				}
			}
		}
	}
	return missing
}

// readPayload walks the payload directory (`data`) and populates bag.payload. F
// File paths are noramilzed with encodePath
func (bag *Bag) readPayload() error {
	bag.payload = Payload{}
	return bag.Backend.Walk(dataDir, func(path string, size int64, err error) error {
		if err == nil {
			encPath := encodePath(path)
			if _, exists := bag.payload[encPath]; exists {
				return fmt.Errorf("path encoding collision: %s", path)
			}
			bag.payload[encPath] = PayloadEntry{rawPath: path, size: size}
		}
		return err
	})
}

// read and parse manifest file with the given name
func (bag *Bag) readManifest(name string) (*Manifest, error) {
	manifest, err := newManifestFromFilename(name)
	if err != nil {
		return nil, err
	}
	return manifest, bag.parse(manifest, name, bag.encoding)
}

// read and pars all manifests (both payload and tag manifests)
func (bag *Bag) readAllManifests() error {
	for _, manName := range bag.Backend.AllManifests() {
		man, err := bag.readManifest(manName)
		if err != nil {
			return err
		}
		switch man.kind {
		case payloadManifest:
			bag.manifests = append(bag.manifests, man)
		case tagManifest:
			bag.tagManifests = append(bag.tagManifests, man)
		default:
			return fmt.Errorf("Unexpected manifest type: %s", manName)
		}
	}
	return nil
}

// read and parse bagit.txt
func (bag *Bag) readBagitTxt() error {
	var t TagFile
	err := bag.parse(&t, bagitTxt, `UTF-8`)
	if err != nil {
		return err
	}
	vals, err := t.bagitTxtValues()
	if err != nil {
		return err
	}
	bag.encoding = vals.encoding
	bag.version = vals.version
	return nil
}

// read and parse bag-info.txt
func (bag *Bag) readBagInfo() error {
	return bag.parse(&bag.Info, bagInfo, bag.encoding)
}

// read and parse fetch.txt
func (bag *Bag) readFetchFile() error {
	_, err := bag.Backend.Stat(fetchTxt)
	if err != nil {
		return nil // not an error if fetch doesn't exist
	}
	return bag.parse(&bag.fetch, fetchTxt, bag.encoding)
}

// parser is an interface used by all bag components types:
// manigest, tagFile, and Fetch.
type parser interface {
	parse(io.Reader) error
}

// parse is a helper function for parsing compontent files in a bag.
// It wraps the logic opening, decoding, and parsing the bag.
func (bag *Bag) parse(parser parser, name string, encoding string) error {
	reader, err := bag.Backend.Open(name)
	defer reader.Close()
	if err != nil {
		return err
	}
	decodeReader, err := newDecodeReader(reader, encoding)
	if err != nil {
		return err
	}
	err = parser.parse(decodeReader)
	if err != nil {
		err := fmt.Errorf("While parsing %s: %s", name, err.Error())
		return err
	}
	return nil
}

func (bag *Bag) versionOk() bool {
	switch bag.version {
	case [...]int{1, 0}:
	case [...]int{0, 97}:
	case [...]int{0, 96}:
	case [...]int{0, 95}:
	case [...]int{0, 94}:
	case [...]int{0, 93}:
	default:
		return false
	}
	return true
}

func (bag *Bag) WritePayloadManifests() error {
	for _, man := range bag.manifests {
		if err := bag.write(man.Filename(), man); err != nil {
			return err
		}
	}
	return nil
}

func (bag *Bag) WriteTagManifests() error {
	for _, man := range bag.tagManifests {
		man.kind = tagManifest
		if err := bag.write(man.Filename(), man); err != nil {
			return err
		}
	}
	return nil
}

func (bag *Bag) WriteBagitTxt() error {
	return bag.write(bagitTxt, DefaultBagitTxt())
}

func (bag *Bag) WriteBagInfo() error {
	return bag.write(bagInfo, &bag.Info)
}

type bagComponent interface {
	Write(io.Writer) error
}

func (bag *Bag) write(path string, writer bagComponent) (err error) {
	var file io.WriteCloser
	if file, err = bag.Backend.Create(path); err != nil {
		return err
	}
	if err = writer.Write(file); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return nil
}
