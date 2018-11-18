package bago

import (
	"os"
	"runtime"
	"testing"

	"github.com/srerickson/bago/test"
)

func TestCreateBag(t *testing.T) {

	fileContent := map[string][]byte{
		`file1.txt`:      []byte(`this is file 1`),
		`dir1/file2.txt`: []byte(`this is file 2`),
	}

	p := test.TmpDataPath(fileContent)
	defer os.RemoveAll(p)
	opts := &CreateBagOptions{
		SrcDir:     p,
		InPlace:    true,
		Algorithms: []string{`sha512`, `md5`},
		Workers:    runtime.GOMAXPROCS(0),
	}
	var bag *Bag
	var err error
	if bag, err = CreateBag(opts); err != nil {
		t.Error(err)
	}
	if _, err := bag.IsValid(); err != nil {
		t.Error(err)
	}
}
