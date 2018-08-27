package bago

import (
	"bufio"
	"fmt"
	"io"
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

func parseFetch(reader io.Reader) (FetchFile, error) {
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
