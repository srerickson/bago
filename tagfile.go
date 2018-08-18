package bago

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type TagFile struct {
	labels []string
	tags   map[string]string
}

func NewTagFile() *TagFile {
	tf := &TagFile{}
	tf.labels = []string{}
	tf.tags = make(map[string]string)
	return tf
}

func ParseTagFile(path string) (*TagFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	tf := NewTagFile()
	lineNum := 0
	emptyLineRE := regexp.MustCompile(`^\s*$`)
	labelLineRe := regexp.MustCompile(`^([^:\s][^:]*):(.*)`)
	contLineRE := regexp.MustCompile(`^\s+\S+`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		errMsg := fmt.Sprintf("Syntax error %s: %d", path, lineNum)

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

func (tf *TagFile) Print() {
	for _, l := range tf.labels {
		fmt.Printf("%s: %s\n", l, tf.tags[l])
	}
}
