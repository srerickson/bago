package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/srerickson/bago"
)

var (
	validate  bool
	create    bool
	processes int
	profile   bool
	algorithm string
	quiet     bool
	path      string
	outPath   string
	tags      bago.TagFile
)

func init() {
	flag.BoolVar(&create, `create`, false, `create bag`)
	flag.BoolVar(&validate, `validate`, false, `validate bag`)
	flag.StringVar(&algorithm, `algorithm`, `sha512`, `checksum algorithm to use`)
	flag.IntVar(&processes, `processes`, 1, `number of processes to use`)
	flag.BoolVar(&profile, `profile`, false, `use profile`)
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
		opts := bago.CreateBagOptions{
			SrcDir:     path,
			DstPath:    outPath,
			Algorithms: []string{algorithm},
			Workers:    processes,
			InPlace:    outPath == ``,
		}
		_, err := bago.CreateBag(&opts)
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
	} else if profile {
		profile := bago.Profile{}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			handleErr(err)
			os.Exit(1)
		}
		err = json.Unmarshal(data, &profile)
		if err != nil {
			handleErr(err)
			os.Exit(1)
		}
		fmt.Printf("%v\n", profile)
		for k, v := range profile.BagInfo {
			fmt.Printf("%s, %v\n", k, v)
		}
	}
}
