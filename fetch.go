package bago

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

type fetch []fetchEntry
type fetchEntry struct {
	url  string
	size string
	path string
}

func (f *fetch) parse(reader io.Reader) error {
	*f = nil
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
			return fmt.Errorf("Syntax error at line: %d", lineNum)
		}
		entry := fetchEntry{}
		entry.url = strings.Trim(match[1], ` `)
		entry.size = strings.Trim(match[2], ` `)
		match[3] = strings.Trim(match[3], ` `)
		entry.path = filepath.Clean(decodePath(match[3]))
		if strings.HasPrefix(entry.path, `..`) {
			return fmt.Errorf("Out of scope path at line: %d", lineNum)
		}
		*f = append(*f, entry)
	}
	return nil
}
