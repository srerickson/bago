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
	if !quiet {
		fmt.Fprintln(os.Stderr, err)
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
		bag, err := bago.OpenBag(path)
		if err != nil {
			handleErr(err)
			os.Exit(1)
		}
		valid := bag.IsValid(handleErr)
		if !valid {
			os.Exit(1)
		}
		// bag.Print()
	}

}
