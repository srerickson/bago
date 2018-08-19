package bago

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type TagFile struct {
	labels []string
	tags   map[string]string
	hasBOM bool // byte order mark present in tagfile?
}

func NewTagFile() *TagFile {
	tf := &TagFile{}
	tf.labels = []string{}
	tf.tags = make(map[string]string)
	return tf
}

func ParseTags(reader io.Reader) (*TagFile, error) {
	tf := NewTagFile()
	lineNum := 0
	emptyLineRE := regexp.MustCompile(`^\s*$`)
	labelLineRe := regexp.MustCompile(`^([^:\s][^:]*):(.*)`)
	contLineRE := regexp.MustCompile(`^\s+\S+`)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// check for BOM
		if lineNum == 1 {
			line = strings.Map(func(r rune) rune {
				if r == '\uFEFF' {
					tf.hasBOM = true
					return -1
				}
				return r
			}, line)

		}

		errMsg := fmt.Sprintf("Syntax error at line: %d", lineNum)

		// ignore empty lines
		if emptyLineRE.MatchString(line) {
			continue
		}

		// continuation of previous label
		if contLineRE.MatchString(line) {
			l := len(tf.labels)
			if l > 0 {
				prevLabel := tf.labels[l-1]
				tf.tags[prevLabel] += " " + strings.Trim(line, ` `)
				continue
			}
			return nil, errors.New(errMsg)
		}

		// start of a new label/value pair
		match := labelLineRe.FindStringSubmatch(line)
		if len(match) < 3 {
			return nil, errors.New(errMsg)
		}
		label := strings.Trim(match[1], ` `)
		value := strings.Trim(match[2], ` `)
		tf.tags[label] = value
		tf.labels = append(tf.labels, label)
	}
	return tf, nil
}

func ReadTagFile(path string) (*TagFile, error) {
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	tags, err := ParseTags(io.Reader(file))
	if err != nil {
		msg := fmt.Sprintf("While reading %s: %s", path, err.Error())
		return nil, errors.New(msg)
	}
	return tags, err
}

func (tf *TagFile) Print() {
	for _, l := range tf.labels {
		fmt.Printf("%s: %s\n", l, tf.tags[l])
	}
}
