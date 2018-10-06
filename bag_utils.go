package bago

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type CreateBagOptions struct {
	SrcDir     string
	DstPath    string
	InPlace    bool
	Algorithms []string
	Info       TagFile
	Workers    int
}

func OpenBag(path string) (*Bag, error) {
	backend := &FSBag{path: path}
	bag := &Bag{Backend: backend}
	return bag, bag.Hydrate()
}

// Create Bag Creates a new Bag with FSBag backend
func CreateBag(opts *CreateBagOptions) (bag *Bag, err error) {

	var buildDir string

	if opts.Workers < 1 {
		opts.Workers = 1
	}
	// set path options to absolute paths
	for _, p := range [2]*string{&opts.SrcDir, &opts.DstPath} {
		if *p, err = filepath.Abs(*p); err != nil {
			err = fmt.Errorf("could not determine absolute path for %s", *p)
			return
		}
	}
	if opts.InPlace {
		// Prepare in-place bag creation
		opts.DstPath = opts.SrcDir
		baseDir := filepath.Dir(opts.DstPath)
		dirName := filepath.Base(opts.SrcDir)
		if buildDir, err = ioutil.TempDir(baseDir, dirName); err != nil {
			return
		}
		cleanup := func() {
			if err != nil {
				os.RemoveAll(buildDir)
			}
		}
		defer cleanup()

	} else {
		// Prepare bag to new destination
		var dstInfo os.FileInfo
		if dstInfo, err = os.Stat(opts.DstPath); err != nil {
			// if opts.DstPath doesn't exist, try to create it
			if err = os.Mkdir(opts.DstPath, 0755); err != nil {
				return
			}
		}
		// if DstPath exists, treat it as the parent of the new bag directory
		if !dstInfo.IsDir() {
			err = fmt.Errorf("expected a directory: %s", opts.DstPath)
			return
		}
		opts.DstPath = filepath.Join(opts.DstPath, filepath.Base(opts.SrcDir))
		if err = os.Mkdir(opts.DstPath, 0755); err != nil {
			return
		}
		buildDir = opts.DstPath
	}

	// TMP Backend is just used to create the initial payload
	tmpBE := &FSBag{path: opts.SrcDir}
	// newBag := &Bag{version: [2]int{1, 0}, encoding: `UTF-8`}
	manifests := map[string]*Manifest{}

	checksumQueue := make(chan checksumJob)
	checksumOutput := checksumWorkers(opts.Workers, checksumQueue, tmpBE)

	go func(algs []string) {
		defer close(checksumQueue)
		tmpBE.Walk(`.`, func(p string, size int64, err error) error {
			for _, alg := range opts.Algorithms {
				checksumQueue <- checksumJob{path: p, alg: alg, err: err}
			}
			fmt.Println(p)
			return err
		})
	}(opts.Algorithms)

	for ch := range checksumOutput {
		if ch.err != nil {
			return nil, ch.err
		}
		manifest, ok := manifests[ch.alg]
		if !ok {
			manifests[ch.alg] = &Manifest{algorithm: ch.alg}
			manifest = manifests[ch.alg]
		}
		err = manifest.Append(filepath.Join(`data`, ch.path), ch.currentSum)
		if err != nil {
			return nil, err
		}
	}

	//Write payload manifests
	var file *os.File
	for _, manifest := range manifests {
		file, err = os.Create(filepath.Join(buildDir, manifest.Filename()))
		if err != nil {
			return nil, err
		}
		manifest.Write(file)
		file.Close()
	}

	//bagit.txt
	if file, err = os.Create(filepath.Join(buildDir, `bagit.txt`)); err != nil {
		return nil, err
	}
	DefaultBagitTxt().Write(file)
	file.Close()

	//bag-info.txt
	opts.Info.Set("Bagging-Date", `today`)
	if file, err = os.Create(filepath.Join(buildDir, `bag-info.txt`)); err != nil {
		return nil, err
	}
	opts.Info.Write(file)
	file.Close()

	//mv or cp?

	return nil, nil

}
