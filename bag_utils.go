package bago

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type CreateBagOptions struct {
	SrcDir     string
	InPlace    bool
	DstPath    string
	Algorithms []string
	Info       TagFile
	Workers    int
	tmpDir     string
}

func OpenBag(path string) (*Bag, error) {
	backend := &FSBag{path: path}
	bag := &Bag{Backend: backend}
	return bag, bag.Hydrate()
}

// Create Bag Creates a new Bag with FSBag backend
func CreateBag(opts *CreateBagOptions) (bag *Bag, err error) {
	if opts.Workers < 1 {
		opts.Workers = 1
	}
	// set paths to absolute paths
	for _, p := range [2]*string{&opts.SrcDir, &opts.DstPath} {
		*p, err = filepath.Abs(*p)
		if err != nil {
			err = fmt.Errorf("could not determine absolute path for %s", *p)
			return
		}
	}
	if opts.InPlace {
		// tmp directory
		opts.DstPath = opts.SrcDir
		baseDir := filepath.Dir(opts.DstPath)
		dirName := filepath.Base(opts.SrcDir)
		if opts.tmpDir, err = ioutil.TempDir(baseDir, dirName); err != nil {
			return
		}
	} else {
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
	}

	// TMP Backend is just used to create the initial payload
	tmpBE := &FSBag{path: opts.SrcDir}
	// newBag := &Bag{version: [2]int{1, 0}, encoding: `UTF-8`}
	manifests := map[string]Manifest{}
	checksumQueue := make(chan checksumJob)
	checksumOutput := checksumWorkers(opts.Workers, checksumQueue, tmpBE)

	go func(algs []string) {
		defer close(checksumQueue)
		tmpBE.Walk(`.`, func(p string, size int64, err error) error {
			for _, alg := range opts.Algorithms {
				checksumQueue <- checksumJob{path: p, alg: alg, err: err}
			}
			return err
		})
	}(opts.Algorithms)
	for ch := range checksumOutput {
		if ch.err != nil {
			return nil, ch.err
		}
		manifest, ok := manifests[ch.alg]
		if !ok {
			manifests[ch.alg] = Manifest{}
			manifest = manifests[ch.alg]
		}

		err = manifest.Append(filepath.Join(`data`, ch.path), ch.currentSum)
		if err != nil {
			return nil, err
		}
		fmt.Println(ch.currentSum)

	}

	return nil, nil

}
