package bago

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
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
	if opts.InPlace { // Prepare in-place bag creation
		opts.DstPath = opts.SrcDir
		baseDir := filepath.Dir(opts.DstPath)
		dirName := filepath.Base(opts.SrcDir)
		if buildDir, err = ioutil.TempDir(baseDir, dirName); err != nil {
			return
		}
	} else { // Prepare bag to new destination
		var dstInfo os.FileInfo
		if dstInfo, err = os.Stat(opts.DstPath); err != nil {
			if err = os.Mkdir(opts.DstPath, 0755); err != nil {
				return
			}
		}
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
	defer func() {
		if err != nil {
			os.RemoveAll(buildDir)
		}
	}()

	// the new bag
	bag = &Bag{
		Backend: &FSBag{path: buildDir},
		Info:    opts.Info,
	}
	bag.manifests, err = ManfifestsForDir(opts.SrcDir, opts.Algorithms, opts.Workers, `data/`)
	if err != nil {
		return nil, err
	}
	if err = bag.WritePayloadManifests(); err != nil {
		return nil, err
	}
	if err = bag.WriteBagitTxt(); err != nil {
		return nil, err
	}
	bag.Info.Set(`Bag-Date`, time.Now().Format("2006-01-02"))
	bag.Info.Set(`Long-Text-Entry`, `This is very very long text that should trigger the line wrap functions. Hope it works!`)
	if err = bag.WriteBagInfo(); err != nil {
		return nil, err
	}
	bag.tagManifests, err = ManfifestsForDir(buildDir, opts.Algorithms, opts.Workers, ``)
	if err != nil {
		return nil, err
	}
	if err = bag.WriteTagManifests(); err != nil {
		return nil, err
	}
	if err = os.Rename(opts.SrcDir, filepath.Join(buildDir, `data`)); err != nil {
		return nil, err
	}
	if opts.InPlace {
		if err = os.Rename(buildDir, opts.DstPath); err != nil {
			return nil, err
		}
	}
	return bag, nil
}

// Manifests for Dir returns a slice of manifests describing contents of a
// directory.
func ManfifestsForDir(dPath string, algs []string, numWorkers int, prefix string) ([]*Manifest, error) {
	if len(algs) == 0 {
		return nil, fmt.Errorf("Can't make manifest without an algorithm")
	}
	for i := range algs {
		var err error
		if algs[i], err = NormalizeAlgName(algs[i]); err != nil {
			return nil, err
		}
	}
	fs := &FSBag{path: dPath}
	mans := map[string]*Manifest{}
	sumer := NewChecksumer(numWorkers, fs)
	go func() {
		defer close(sumer.jobs)
		walkErr := fs.Walk(`.`, func(p string, s int64, err error) error {
			for _, alg := range algs {
				sumer.Push(p, alg, ``)
			}
			return err
		})
		if walkErr != nil {
			sumer.jobs <- checksumJob{path: dPath, alg: ``, err: walkErr}
		}
	}()
	for check := range sumer.Results() {
		if check.err != nil {
			return nil, check.err
		}
		_, ok := mans[check.alg]
		if !ok {
			mans[check.alg] = &Manifest{algorithm: check.alg}
		}
		err := mans[check.alg].Append(prefix+check.path, check.actualSum)
		if err != nil {
			return nil, err
		}
	}
	ret := make([]*Manifest, len(algs))
	for i, alg := range algs {
		ret[i] = mans[alg]
	}
	return ret, nil
}
