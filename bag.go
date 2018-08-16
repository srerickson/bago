package bago

import (
	"errors"
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
	manifests []Manifest
	persisted bool
}

// NewBag returns a new bag ()
func NewBag(path string) *Bag {
	bag := &Bag{path: path, persisted: false}
	return bag
}

// IsComplete returns whether bag satisfies bag completeness conditions.
// See: https://tools.ietf.org/html/draft-kunze-bagit-16#section-3
func (b *Bag) IsComplete() (bool, error) {

	_, err := os.Stat(filepath.Join(b.path, "bagit.txt"))
	if err != nil {
		return false, err
	}

	file, err := os.Stat(filepath.Join(b.path, "data"))
	if !file.Mode().IsDir() {
		return false, errors.New("missing data directory")
	}
	if err != nil {
		return false, err
	}

	manifests, err := filepath.Glob(filepath.Join(b.path, "manifest-*.txt"))
	if len(manifests) == 0 {
		return false, errors.New("missing manifest")
	}
	if err != nil {
		return false, err
	}

	// manifest, errs := ReadManifest(manifests[0])
	// if errs != nil {
	// 	for e := range errs {
	// 		fmt.Println(errs[e])
	// 	}
	// }

	return true, nil
}

// IsValid returns whether the bag at path is valid
func (b *Bag) IsValid() (bool, error) {

	// var wg sync.WaitGroup

	// valid bags are complete
	complete, err := b.IsComplete()
	if !complete {
		if err == nil {
			panic("bag is incomplete but without errors!")
		}
		return false, err
	}

	payloadDir := filepath.Join(b.path, "data")
	GenerateManifest(payloadDir, 8, `sha512`)

	// filepath.Walk(b.path, func(path string, info os.FileInfo, err error) error {
	// 	fmt.Println(path)
	// 	if err != nil {
	// 		fmt.Printf("error accessing a path %q: %v\n", path, err)
	// 		return err
	// 	}
	// 	if info.Mode().IsDir() {
	// 		return nil
	// 	}
	// 	// sum, _ := Checksum(p, `sha512`)
	// 	// fmt.Printf("%s: %s\n", path, sum)
	// 	wg.Add(1)
	// 	go func(p string) {
	// 		defer wg.Done()
	// 		sum, _ := Checksum(p, `sha512`)
	// 		fmt.Printf("%s: %s\n", p, sum)
	// 	}(path)
	//
	// 	return nil
	// })
	return true, nil
}
