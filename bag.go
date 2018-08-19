package bago

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const (
	defaultVersion = "1.0" // BagIt version for new bags
	bagitTxt       = `bagit.txt`
	dataDir        = `data`
)

// Bag is a bagit repository
type Bag struct {
	path         string
	version      string
	manifests    []*Manifest
	tagManifests []*Manifest
	tagFiles     map[string]*TagFile
	payload      *Manifest
}

// LoadBag returs Bag object for bag at path
func LoadBag(path string) (*Bag, error) {
	bag := &Bag{}
	bag.path = path
	// data payload present?
	dataInfo, err := os.Stat(filepath.Join(path, dataDir))
	if err != nil {
		return nil, err
	}
	if !dataInfo.IsDir() {
		return nil, errors.New(`no data directory`)
	}
	// read bagit.txt
	bagitTags, err := ReadTagFile(filepath.Join(path, bagitTxt))
	if err != nil {
		return nil, err
	}
	bag.tagFiles = make(map[string]*TagFile)
	bag.tagFiles[bagitTxt] = bagitTags
	// read manifests for both payload and tag files
	mans, err := ReadAllManifests(path)
	if err != nil {
		return nil, err
	}
	for i := range mans {
		if mans[i].kind == payloadManifest {
			bag.manifests = append(bag.manifests, mans[i])
		} else if mans[i].kind == tagManifest {
			bag.tagManifests = append(bag.tagManifests, mans[i])
		}
	}
	if len(bag.manifests) == 0 {
		return nil, errors.New(`no manifests found`)
	}
	return bag, nil
}

// IsComplete returns whether bag satisfies bag completeness conditions.
// See: https://tools.ietf.org/html/draft-kunze-bagit-16#section-3
func (b *Bag) IsComplete() (bool, error) {
	// check bagit.txt
	err := b.bagitTxtErrors()
	if err != nil {
		return false, err
	}

	// build payload
	if b.payload == nil {
		b.payload = NewManifest(``)
	}
	fileChan := fileWalker(filepath.Join(b.path, dataDir))
	for f := range fileChan {
		relPath, _ := filepath.Rel(b.path, f.path) // file path relative to bag
		encPath := encodePath(relPath)             // encoded for manifests
		// check if already exists?
		entry := ManifestEntry{
			rawPath: f.path,
			size:    f.info.Size(),
		}
		// record which manifests this file was not found in
		for _, m := range b.manifests {
			if _, ok := m.entries[encPath]; !ok {
				entry.notIn = append(entry.notIn, m)
			}
		}
		b.payload.entries[encPath] = entry
	}
	missing := []string{}
	for p, e := range b.payload.entries {
		if len(e.notIn) > 0 {
			missing = append(missing, p)
		}
	}

	if len(missing) > 0 {
		return false, errors.New(fmt.Sprintf("Missing file in manifest: %s", missing[0]))
	}

	return true, nil

}

// IsValid returns whether the bag at path is valid
func (b *Bag) IsValid() (bool, error) {
	_, err := b.IsComplete()
	if err != nil {
		return false, err
	}
	return true, nil
}

// Print Bag Contents
func (b *Bag) Print() {
	fmt.Println(b.path)
	for i := range b.manifests {
		fmt.Println(b.manifests[i].algorithm)
		for path, e := range b.manifests[i].entries {
			fmt.Printf("-- %s: %s\n", path, e.sum)
		}
	}
}

func (b *Bag) bagitTxtErrors() error {
	required := map[string]*regexp.Regexp{
		`BagIt-Version`:               regexp.MustCompile(`(\d+)\.(\d+)`),
		`Tag-File-Character-Encoding`: regexp.MustCompile(`(.*)`),
	}
	bagit := b.tagFiles[bagitTxt]
	if bagit == nil {
		return errors.New(`Missing bagit.txt`)
	}
	if bagit.hasBOM {
		msg := fmt.Sprintf(`%s has BOM`, bagitTxt)
		return errors.New(msg)
	}
	for label, pattern := range required {
		if bagit.tags[label] == `` {
			msg := fmt.Sprintf(`Required field missing in %s: %s`, bagitTxt, label)
			return errors.New(msg)
		}
		if !pattern.MatchString(bagit.tags[label]) {
			msg := fmt.Sprintf(`Malformed value in %s for label %s: %s`, bagitTxt, label, bagit.tags[label])
			return errors.New(msg)
		}
	}
	return nil
}
