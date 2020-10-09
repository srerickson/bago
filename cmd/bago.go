package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/integrii/flaggy"
	"github.com/srerickson/bago"
	"github.com/srerickson/bago/checksum"
)

const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
)

var redErr = colorRed + "[ERR]" + colorReset
var greenOK = colorGreen + "[OK]" + colorReset

var version = "unknown"
var subCmd = make(map[string]*flaggy.Subcommand)

// default parameters
var processes = runtime.GOMAXPROCS(0)
var verbose = false
var algorithms = []string{checksum.SHA512}
var path = `./`
var outPath = ``
var tags = []string{}

func init() {
	flaggy.SetName("bago")
	flaggy.SetDescription("Command Line Tool for creating and validating Bag-It Bags")

	// global flags
	flaggy.Int(&processes, `p`, `procs`, `number of goroutines allocated for checksum`)
	flaggy.Bool(&verbose, `v`, `verbose`, `verbose validation`)

	// validate subcommand
	subCmd[`validate`] = flaggy.NewSubcommand("validate")
	subCmd[`validate`].Description = "Validate a Bag"
	subCmd[`validate`].AddPositionalValue(&path, `path`, 1, true, `bag to validate`)

	// create subcommand
	subCmd[`create`] = flaggy.NewSubcommand("create")
	subCmd[`create`].Description = "Create a Bag"
	subCmd[`create`].AddPositionalValue(&path, `path`, 1, true, `folder to bag`)
	subCmd[`create`].String(&outPath, `o`, `output`, `destination for new bag`)
	subCmd[`create`].StringSlice(&algorithms, `a`, `algs`, `checksum algorithms`)

	for i := range subCmd {
		flaggy.AttachSubcommand(subCmd[i], 1)
	}
	flaggy.SetVersion(version)
	flaggy.Parse()
}

func main() {

	if subCmd[`create`].Used {
		opts := bago.CreateBagOptions{
			SrcDir: path,
			// Info:       bago.TagFile(tags),
			DstPath:    outPath,
			Algorithms: algorithms,
			Workers:    processes,
			InPlace:    outPath == ``,
		}
		_, err := bago.CreateBag(&opts)
		if err != nil {
			log.Fatalf(`Could not create bag: %s`, err.Error())
		}
		fmt.Println(`Created new bag`)
	}

	if subCmd[`validate`].Used {
		bag, err := bago.OpenBag(path)
		if err != nil {
			log.Fatalf(`%s Not a bag: %s`, redErr, path)
		}
		if _, err := bag.IsValidConcurrent(processes); err != nil {
			if verbose {
				log.Fatalf("%s Bag is invalid: %s\n Errors:%s", redErr, path, err.Error())
				return
			}
			log.Fatalf("%s Bag is invalid: %s", redErr, path)
		}
		log.Printf("%s Bag is valid: %s", greenOK, path)
	}

	// } else if profile {
	// 	profile := bago.Profile{}
	// 	data, err := ioutil.ReadFile(path)
	// 	if err != nil {
	// 		handleErr(err)
	// 		os.Exit(1)
	// 	}
	// 	err = json.Unmarshal(data, &profile)
	// 	if err != nil {
	// 		handleErr(err)
	// 		os.Exit(1)
	// 	}
	// 	fmt.Printf("%v\n", profile)
	// 	for k, v := range profile.BagInfo {
	// 		fmt.Printf("%s, %v\n", k, v)
	// 	}
	// }
}
