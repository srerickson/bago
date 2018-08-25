package bago

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type FetchFile []Fetch
type Fetch struct {
	url  string
	size string
	path string
}

func ParseFetch(reader io.Reader) (FetchFile, error) {
	fFile := FetchFile{}
	lineNum := 0
	emptyLineRE := regexp.MustCompile(`^\s*$`)
	fetchRE := regexp.MustCompile(`^(\S+)\s+(\S+)\s+(.*)$`)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if emptyLineRE.MatchString(line) {
			continue // ignore empty lines
		}
		match := fetchRE.FindStringSubmatch(line)
		if len(match) < 4 {
			return nil, fmt.Errorf("Syntax error at line: %d", lineNum)
		}
		f := Fetch{}
		f.url = strings.Trim(match[1], ` `)
		f.size = strings.Trim(match[2], ` `)
		match[3] = strings.Trim(match[3], ` `)
		f.path = filepath.Clean(decodePath(match[3]))
		if strings.HasPrefix(f.path, `..`) {
			return nil, fmt.Errorf("Out of scope path at line: %d", lineNum)
		}
		fFile = append(fFile, f)

	}
	return fFile, nil
}

func ReadFetchFile(path string, enc string) (FetchFile, error) {
	_, err := os.Stat(path)
	if err != nil {
		// not an error if fetch doesn't exist
		return FetchFile{}, nil
	}
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	decodeReader, err := newReader(file, enc)
	if err != nil {
		return nil, err
	}
	fetch, err := ParseFetch(decodeReader)
	if err != nil {
		msg := fmt.Sprintf("While reading %s: %s", path, err.Error())
		return nil, errors.New(msg)
	}
	return fetch, err
}
