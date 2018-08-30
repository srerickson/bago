package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/srerickson/bago"
)

var (
	validate  bool
	create    bool
	processes int
	algorithm string
	quiet     bool
	path      string
	outPath   string
	tags      bago.TagFile
)

func init() {
	flag.BoolVar(&create, `create`, false, `create bag`)
	flag.IntVar(&processes, `processes`, 1, `number of processes to use`)
	flag.StringVar(&algorithm, `algorithm`, `sha512`, `checksum algorithm to use`)
	flag.BoolVar(&validate, `validate`, false, `validate bag`)
	flag.BoolVar(&quiet, `quiet`, false, `no ouput (on STDOUT)`)
	flag.StringVar(&outPath, `o`, ``, `output path`)
	flag.Var(&tags, `t`, `set tag`)
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
	if create {
		if outPath == `` {
			outPath = path
		}
		_, err := bago.CreateBag(path, outPath, algorithm, processes)
		if err != nil {
			handleErr(err)
			os.Exit(1)
		}
	} else if validate {
		bag, err := bago.OpenBag(path)
		if err != nil {
			handleErr(err)
			os.Exit(1)
		}
		valid := bag.IsValid(handleErr)
		if !valid {
			os.Exit(1)
		}
		// fmt.Println(bag.Info.String())
	}

}
