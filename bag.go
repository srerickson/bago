package bago

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultVersion = "1.0" // BagIt version for new bags
)

// Bag is a bagit repository
type Bag struct {
	path      string
	version   string
	manifests []*Manifest
	tagFiles  map[string]*TagFile
}

// LoadBag returs Bag object for bag at path
func LoadBag(path string) (*Bag, error) {
	bag := &Bag{}
	bag.path = path

	// data payload
	dataDir := filepath.Join(path, `data`)
	dataInfo, err := os.Stat(dataDir)
	if err != nil {
		return nil, err
	}
	if !dataInfo.IsDir() {
		return nil, errors.New(`'data' is not a directory`)
	}

	bagitTags, err := ParseTagFile(filepath.Join(path, `bagit.txt`))
	if err != nil {
		return nil, err
	}
	bag.tagFiles = make(map[string]*TagFile)
	bag.tagFiles[`bagit.txt`] = bagitTags
	// bag.version = bag.tagFiles[`bagit.txt`].tags[`BagIt-Version`]

	mans, err := ParseAllManifests(path)
	if err != nil {
		return bag, err
	}
	bag.manifests = mans

	return bag, nil

}

// IsComplete returns whether bag satisfies bag completeness conditions.
// See: https://tools.ietf.org/html/draft-kunze-bagit-16#section-3
func (b *Bag) IsComplete() (bool, error) {
	return false, nil
}

// IsValid returns whether the bag at path is valid
func (b *Bag) IsValid() (bool, error) {
	return true, nil
}

// Print Bag Contents
func (b *Bag) Print() {
	fmt.Println(b.path)
	for i := range b.manifests {
		fmt.Println(b.manifests[i].algorithm)
		for path, sum := range b.manifests[i].entries {
			fmt.Printf("-- %s: %s\n", path, sum)
		}
	}
}
