package bago

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/srerickson/bago/backend"
	"github.com/srerickson/bago/checksum"
)

// CreateBagOptions is used to pass options to CreateBag()
type CreateBagOptions struct {
	SrcDir     string
	DstPath    string
	InPlace    bool
	Algorithms []string
	Info       TagFile
	Workers    int
}

func OpenBag(path string) (*Bag, error) {
	bag := &Bag{Backend: &backend.FS{Path: path}}
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
	//DstPath can't be a subdir of SrcDir
	var rel string
	if rel, err = filepath.Rel(opts.SrcDir, opts.DstPath); err != nil {
		return
	}
	if !strings.HasPrefix(rel, `..`) {
		err = fmt.Errorf("%s is a subdirectory of %s", opts.DstPath, opts.SrcDir)
		return
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
			// If dstPath doesn't exist, try to create it
			if !os.IsNotExist(err) {
				return
			}
			if err = os.Mkdir(opts.DstPath, 0755); err != nil {
				return
			}
		} else {
			// If dstPath exists, treat as parent dir for new bag
			if !dstInfo.IsDir() {
				err = fmt.Errorf("expected a directory: %s", opts.DstPath)
				return
			}
			opts.DstPath = filepath.Join(opts.DstPath, filepath.Base(opts.SrcDir))
			if err = os.Mkdir(opts.DstPath, 0755); err != nil {
				return
			}
		}
		buildDir = opts.DstPath
	}
	defer func() {
		if err != nil {
			os.RemoveAll(buildDir)
		}
	}()

	// tmp Bag
	bag = &Bag{
		Backend:  &backend.FS{Path: buildDir},
		Info:     opts.Info,
		encoding: `UTF-8`,
		version:  [...]int{0, 97},
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
	bag.Info.Set(`Bag-Software-Agent`, `bago`)
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
	bag, err = OpenBag(opts.DstPath)
	return
}

// Manifests for Dir returns a slice of manifests describing contents of a
// directory.
func ManfifestsForDir(dPath string, algs []string, numWorkers int, prefix string) ([]*Manifest, error) {
	if len(algs) == 0 {
		return nil, fmt.Errorf("Can't make manifest without an algorithm")
	}
	for i := range algs {
		var err error
		if algs[i], err = checksum.NormalizeAlgName(algs[i]); err != nil {
			return nil, err
		}
	}
	fs := &backend.FS{Path: dPath}
	mans := map[string]*Manifest{}
	sumer := checksum.New(numWorkers, fs, func(push checksum.JobPusher) error {
		return fs.Walk(`.`, func(p string, fi os.FileInfo, err error) error {
			for _, alg := range algs {
				push(checksum.Job{Path: p, Alg: alg, Err: err})
			}
			return err
		})
	})
	for check := range sumer.Results() {
		_, ok := mans[check.Alg]
		if !ok {
			mans[check.Alg] = &Manifest{algorithm: check.Alg}
		}
		err := mans[check.Alg].Append(prefix+check.Path, check.SumString())
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
