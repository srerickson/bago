package main

import (
	"flag"
	"os"

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
		b := bago.NewBag(path)
		b.IsValid()
	}
}
