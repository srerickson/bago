package bago

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultVersion = "1.0" // BagIt version for new bags
	bagitTxt       = `bagit.txt`
	bagInfo        = `bag-info.txt`
	dataDir        = `data`
)

// Bag is a bagit repository
type Bag struct {
	path         string
	version      string
	encoding     string
	payload      *Payload
	bagInfo      *TagFile
	manifests    []*Manifest
	tagManifests []*Manifest
}

// LoadBag returs Bag object for bag at path
func LoadBag(path string) (*Bag, error) {
	bag := &Bag{path: path}
	// read bagit.txt
	bagitTags, err := ReadTagFile(filepath.Join(path, bagitTxt), `UTF-8`)
	if err != nil {
		return bag, err
	}
	bag.version, bag.encoding, err = getBagitTxtValues(bagitTags)
	if err != nil {
		return bag, err
	}
	// try to load bag-info.txt
	bag.bagInfo, _ = ReadTagFile(filepath.Join(path, bagInfo), bag.encoding)
	// load payload
	bag.payload, err = loadPayload(bag.path)
	if err != nil {
		return bag, err
	}
	// read manifests for both payload and tag files
	mans, err := ReadAllManifests(path, bag.encoding)
	if err != nil {
		return bag, err
	}
	for i := range mans {
		if mans[i].kind == payloadManifest {
			bag.manifests = append(bag.manifests, mans[i])
		} else if mans[i].kind == tagManifest {
			bag.tagManifests = append(bag.tagManifests, mans[i])
		}
	}
	if len(bag.manifests) == 0 {
		return bag, errors.New(`no manifests found`)
	}
	return bag, nil
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
					path:        filepath.Join(b.path, decodePath(path)),
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

func (b *Bag) missingTagFiles() []string {
	missing := []string{}
	for _, m := range b.tagManifests {
		for tPath, tEntry := range m.entries {
			_, err := os.Stat(filepath.Join(b.path, tEntry.rawPath))
			if err != nil {
				missing = append(missing, tPath)
			}
		}
	}
	return missing
}

func (b *Bag) missingFromPayload() []string {
	missing := []string{}
	for _, m := range b.manifests {
		for mPath := range m.entries {
			if _, ok := b.payload.entries[mPath]; !ok {
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
	for pPath := range b.payload.entries {
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

//
// func (b *Bag) Encoding() (string, error) {
// 	if b.encoding != `` {
// 		return b.encoding, nil
// 	}
// 	return ``, fmt.Errorf("Encoding not defined")
// }
