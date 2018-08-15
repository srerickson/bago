package bago

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Bag is a bagit repository
type Bag struct {
	path      string
	version   string
	manifests []Manifest
}

// IsComplete returns whether bag satisfies bag completeness conditions.
// See: https://tools.ietf.org/html/draft-kunze-bagit-16#section-3
func IsComplete(path string) (bool, error) {

	_, err := os.Stat(filepath.Join(path, "bagit.txt"))
	if err != nil {
		return false, err
	}

	file, err := os.Stat(filepath.Join(path, "data"))
	if !file.Mode().IsDir() {
		return false, errors.New("missing data directory")
	}
	if err != nil {
		return false, err
	}

	manifests, err := filepath.Glob(filepath.Join(path, "manifest-*.txt"))
	if len(manifests) == 0 {
		return false, errors.New("missing manifest")
	}
	if err != nil {
		return false, err
	}

	manifest, errs := LoadManifest(manifests[0])
	if errs != nil {
		for e := range errs {
			fmt.Println(errs[e])
		}
	}
	fmt.Println(len(manifest.entries))

	return true, nil
}

func validation(path string) error {

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		fmt.Println(file.Name())
	}

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("error accessing a path %q: %v\n", p, err)
			return err
		}
		rel, _ := filepath.Rel(path, p)
		fmt.Printf("visited file: %q\n", rel)
		return nil
	})
	return nil
}
