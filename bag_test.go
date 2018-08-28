package bago

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

type TestBagGroup map[string]TestVersionGroup

type TestVersionGroup struct {
	valid   map[string]string
	invalid map[string]string
}

func testDataPath() string {
	_, fPath, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(fPath), `test-data`)
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
	filepath.Walk(testDataPath(), walker)
	return bags
}

func TestOpenBag(t *testing.T) {

	_, err := OpenBag(filepath.Join(testDataPath(), `nobaghere`))
	if err == nil {
		t.Error("Expected an error got", err)
	}
}

func TestIsValid(t *testing.T) {
	wasCalled := false
	callBack := func(e error) {
		wasCalled = true
	}
	for version, group := range testBags() {
		for name, path := range group.valid {
			bag, _ := OpenBag(path)
			wasCalled = false
			isValid := bag.IsValid(callBack)
			if !isValid {
				t.Error("Valid test bag should be valid:", version, name)
			} else if wasCalled {
				t.Error("callback was called during IsValid, but shouldn't have")
			}
		}
		for name, path := range group.invalid {
			bag, _ := OpenBag(path)
			wasCalled = false
			isValid := bag.IsValid(callBack)
			if isValid {
				t.Error("Invalid test bag should be invalid:", version, name)
			} else if !wasCalled {
				t.Error("callback was not called during IsValid, but should have been")
			}
		}
	}
}
