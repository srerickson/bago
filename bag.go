package bago

import (
	"errors"
	"fmt"
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
	Backend      Backend     // backend interface
	version      string      // from bagit.txt
	encoding     string      // from bagit.txt
	payload      Payload     // contents of the data directory
	bagInfo      *TagFile    // contents of bag-info.txt
	manifests    []*Manifest // list of payload manifests
	tagManifests []*Manifest // list of tag file manifests
	fetch        FetchFile   // contents of fetch.txt
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
	bagitTags, err := bag.readTagFile(bagitTxt, `UTF-8`)
	if err != nil {
		return err
	}
	bag.version, bag.encoding, err = getBagitTxtValues(bagitTags)
	if err != nil {
		return err
	}
	bag.bagInfo, _ = bag.readTagFile(bagInfo, bag.encoding)
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
	if len(bag.manifests) == 0 {
		return errors.New(`no manifests found`)
	}
	return nil
}

// IsComplete returns whether bag satisfies bag completeness conditions.
// See: https://tools.ietf.org/html/draft-kunze-bagit-16#section-3
func (b *Bag) IsComplete(errCb func(error)) bool {
	complete := true
	if b.version == `` && b.encoding == `` {
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
func (b *Bag) IsValid(errCb func(error)) bool {
	valid := b.IsComplete(errCb)
	if !valid {
		return false
	}
	// queue up checksum jobs
	jobInput := make(chan checksumJob)
	jobOutput := checksumWorkers(2, jobInput)
	go func() {
		defer close(jobInput)
		// checksums for all manifest entries
		for _, m := range append(b.manifests, b.tagManifests...) {
			for path, entry := range m.entries {
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

func (bag *Bag) readManifest(name string) (*Manifest, error) {
	file, err := bag.Backend.Open(name)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	manifest, err := newManifestFromFilename(name)
	if err != nil {
		return nil, err
	}
	decodeReader, err := newReader(file, bag.encoding)
	if err != nil {
		return nil, err
	}
	return manifest, manifest.parseEntries(decodeReader)
}

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
		}
	}
	return nil
}

func (bag *Bag) readTagFile(name string, encoding string) (*TagFile, error) {
	reader, err := bag.Backend.Open(name)
	defer reader.Close()
	if err != nil {
		return nil, err
	}
	decodeReader, err := newReader(reader, encoding)
	if err != nil {
		return nil, err
	}
	tags, err := ParseTags(decodeReader)
	if err != nil {
		err := fmt.Errorf("While reading %s: %s", name, err.Error())
		return nil, err
	}
	return tags, err
}

func (bag *Bag) readFetchFile() error {
	_, err := bag.Backend.Stat(fetchTxt)
	if err != nil {
		return nil // not an error if fetch doesn't exist
	}
	file, err := bag.Backend.Open(fetchTxt)
	defer file.Close()
	if err != nil {
		return err
	}
	decodeReader, err := newReader(file, bag.encoding)
	if err != nil {
		return err
	}
	bag.fetch, err = parseFetch(decodeReader)
	if err != nil {
		return fmt.Errorf("While reading %s: %s", fetchTxt, err.Error())
	}
	return nil
}
