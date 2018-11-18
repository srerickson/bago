package test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// TmpDataPath builds a tmpdir with test content for bagging and retuns
// its path
func TmpDataPath(fileContent map[string][]byte) string {
	var err error
	var path string
	var checkErr = func(err error) {
		if err == nil {
			return
		}
		if path != `` {
			os.RemoveAll(path)
		}
		log.Fatal(err)
	}
	path, err = ioutil.TempDir(``, `testBagData`)
	checkErr(err)
	for f, c := range fileContent {
		f = filepath.FromSlash(f)
		if d := filepath.Dir(f); d != `.` {
			checkErr(os.MkdirAll(filepath.Join(path, d), 0755))
		}
		checkErr(ioutil.WriteFile(filepath.Join(path, f), c, 0644))
	}
	return path
}

// Returns absolute path to test data
func Path(relPath []string) string {
	_, fPath, _, _ := runtime.Caller(0)
	absPath := make([]string, len(relPath)+1)
	absPath[0] = filepath.Dir(fPath)
	for i := range relPath {
		absPath[i+1] = relPath[i]
	}
	return filepath.Join(absPath...)
}
