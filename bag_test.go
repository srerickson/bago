package bago

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/srerickson/bago/test"
)

type TestBagGroup map[string]TestVersionGroup

type TestVersionGroup struct {
	valid   map[string]string
	invalid map[string]string
}

func testBags() TestBagGroup {
	bags := TestBagGroup{}
	bagPattern := regexp.MustCompile(`(v[0-1]\.\d+)[\\\/](valid|invalid)[\\\/]([^\\\/]*)$`)
	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return err
		}
		matches := bagPattern.FindStringSubmatch(path)
		if len(matches) < 4 {
			return err
		}
		version, validity, name := matches[1], matches[2], matches[3]
		if _, ok := bags[version]; !ok {
			bags[version] = TestVersionGroup{
				valid:   map[string]string{},
				invalid: map[string]string{},
			}
		}
		switch validity {
		case `valid`:
			bags[version].valid[name] = path
		case `invalid`:
			bags[version].invalid[name] = path
		}
		return err
	}
	filepath.Walk(test.DataPath([]string{`bags`}), walker)
	return bags
}

func TestOpenBag(t *testing.T) {

	_, err := OpenBag(test.DataPath([]string{`bags`, `nobaghere`}))
	if err == nil {
		t.Error("Expected an error got", err)
	}
}

func TestIsValid(t *testing.T) {
	for version, group := range testBags() {
		for name, path := range group.valid {
			bag, _ := OpenBag(path)
			isValid, err := bag.IsValidConcurrent(runtime.GOMAXPROCS(0))
			if !isValid {
				t.Errorf("Valid test bag should be valid (%s, %s): %s", version, name, err.Error())
			}
		}
		for name, path := range group.invalid {
			bag, _ := OpenBag(path)
			isValid, _ := bag.IsValidConcurrent(runtime.GOMAXPROCS(0))
			if isValid {
				t.Errorf("Invalid bag should be invalid (%s, %s)", version, name)
			}
		}
	}
}
