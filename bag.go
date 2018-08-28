package bago

import (
	"errors"
	"fmt"
	"io"
)

const (
	defaultVersion = "1.0" // BagIt version for new bags
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
	bagInfo      TagFile     // contents of bag-info.txt
	manifests    []*Manifest // list of payload manifests
	tagManifests []*Manifest // list of tag file manifests
	fetch        fetch       // contents of fetch.txt
}

type Payload map[string]PayloadEntry

type PayloadEntry struct {
	rawPath string
	size    int64
}

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
	err = bag.initPayload()
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
func (b *Bag) IsComplete(errCb func(error)) bool {
	complete := true
	if b.encoding == `` {
		complete = false
		if errCb != nil {
			errCb(fmt.Errorf("Missing required fields in %s", bagitTxt))
		}
	}
	if b.payload == nil {
		if errCb != nil {
			errCb(fmt.Errorf("bag has no payload"))
		}
		return false
	}
	if len(b.manifests) == 0 {
		if errCb != nil {
			errCb(fmt.Errorf("bag has no manifest"))
		}
		return false
	}
	missingFromPayload := b.missingFromPayload()
	if len(missingFromPayload) > 0 {
		complete = false
		if errCb != nil {
			for _, m := range missingFromPayload {
				errCb(fmt.Errorf("Missing from payload: %s", m))
			}
		}
	}
	missingFromManifests := b.missingFromManifests(0)
	if len(missingFromManifests) > 0 {
		complete = false
		if errCb != nil {
			for _, m := range missingFromManifests {
				errCb(fmt.Errorf("Missing from manifests: %s", m))
			}
		}
	}
	missingTags := b.missingTagFiles()
	if len(missingTags) > 0 {
		complete = false
		if errCb != nil {
			for _, m := range missingTags {
				errCb(fmt.Errorf("Missing tag file: %s", m))
			}
		}
	}
	return complete
}

// IsValid returns whether the bag at path is valid
// A valid bag is complete and checksums listed in all manifests are correct.
func (b *Bag) IsValid(errCb func(error)) bool {
	valid := b.IsComplete(errCb)
	if !valid {
		return false
	}
	// queue up checksum jobs
	jobInput := make(chan checksumJob)
	jobOutput := checksumWorkers(2, jobInput, b.Backend)
	go func() {
		defer close(jobInput)
		for _, m := range append(b.manifests, b.tagManifests...) {
			for path, entry := range m.entries {
				// put checksum jobs for each manifest entries on the worker queue
				jobInput <- checksumJob{
					path:        decodePath(path),
					alg:         m.algorithm,
					expectedSum: entry.sum,
				}
			}
		}
	}()
	for job := range jobOutput {
		if job.expectedSum != job.currentSum {
			valid = false
			if errCb != nil {
				errCb(fmt.Errorf("Checksum failed for: %s", job.path))
			}
		}
	}
	return valid
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

// missingFromPayload scans manifests for files not present in the payload.
func (b *Bag) missingFromPayload() []string {
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

// missingFromManifests scans payload for files not accounted for in manifests
// thesh is the min number of manifests that a file can be missing from
func (b *Bag) missingFromManifests(thresh int) []string {
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

// walk the payload directory (`data`) tree and populate bag.payload with the
// files found there. File paths are noramilzed with encodePath
func (bag *Bag) initPayload() error {
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
			return fmt.Errorf("Unknown manifest type: %s", manName)
		}
	}
	return nil
}

// read and parse bagit.txt
func (bag *Bag) readBagitTxt() error {
	var t TagFile
	err := bag.parse(&t, bagitTxt, `UTF-8`)
	if err != nil {
		return nil
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
	return bag.parse(&bag.bagInfo, bagInfo, bag.encoding)
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
