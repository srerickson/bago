package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/srerickson/bago"
)

var (
	validate bool
	quiet    bool
	path     string
)

func init() {
	flag.BoolVar(&validate, "validate", false, "validate bag")
	flag.BoolVar(&quiet, "quiet", false, "no ouput (on STDOUT)")

}

func handleErr(err error) {
	if err != nil {
		if !quiet {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func main() {
	flag.Parse()
	path = flag.Arg(0)
	if path == `` {
		err := errors.New(`no path given`)
		handleErr(err)
	}
	if validate {
		bag, err := bago.LoadBag(path)
		handleErr(err)
		_, err = bag.IsValid()
		handleErr(err)
		// bag.Print()
	}

}
