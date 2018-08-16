package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/srerickson/bago"
)

var validate bool
var path string

func init() {
	flag.BoolVar(&validate, "validate", false, "validate bag")
}

func main() {
	flag.Parse()
	path = flag.Arg(0)
	if path == "" {
		os.Exit(1)
	}
	if validate {
		// b := bago.NewBag(path)
		// b.IsValid()
		mb := &bago.ManifestBuilder{Path: filepath.Join(path, "data"), Workers: 4, Alg: `sha512`}
		mb.Build()
	}
}
