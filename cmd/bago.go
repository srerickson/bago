package main

import (
	"flag"
	"fmt"
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
		bag, err := bago.LoadBag(path)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		_, err = bag.IsValid()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		} else {
			fmt.Println("valid")
		}
		bag.Print()
	}

}
