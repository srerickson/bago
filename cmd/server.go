package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/srerickson/bago"
)

// GET bags/ -> List Bags
// GET bags/{bagName} -> Status & BagInfo
// GET bags/{bagName}/manifest
// GET bags/{bagName}/stat?file={filePath}

// GET POST staging

func server(root string) {

	ListBagsHandler := func(w http.ResponseWriter, r *http.Request) {
		dirs, err := ioutil.ReadDir(root)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, f := range dirs {
			if f.IsDir() {
				fmt.Fprintf(w, "%v\n", f.Name())
			}
		}
	}

	BagHandler := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bag, err := bago.OpenBag(filepath.Join(root, vars[`bagName`]))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		bag.Info.Write(w)
	}

	r := mux.NewRouter()
	r.HandleFunc("/bags", ListBagsHandler)
	r.HandleFunc("/bags/{bagName}", BagHandler)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
