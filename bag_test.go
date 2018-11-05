package bago

import (
	"os"
	"path/filepath"
	"regexp"
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
			isValid, _ := bag.IsValidConcurrent(20)
			if !isValid {
				t.Error("Valid test bag should be valid:", version, name)
			}
		}
		for name, path := range group.invalid {
			bag, _ := OpenBag(path)
			isValid, _ := bag.IsValid()
			if isValid {
				t.Error("Invalid test bag should be invalid:", version, name)
			}
		}
	}
}
