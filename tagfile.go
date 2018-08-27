package bago

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type TagSet map[string][]string

type TagFile struct {
	tags   TagSet
	labels []string
}

func NewTagFile() *TagFile {
	tf := &TagFile{tags: TagSet{}}
	return tf
}

func (tf *TagFile) Append(label string, value string) []string {
	if _, ok := tf.tags[label]; !ok {
		tf.tags[label] = []string{value}
		tf.labels = append(tf.labels, label)
	} else {
		tf.tags[label] = append(tf.tags[label], value)
	}
	return tf.tags[label]
}

func (tf *TagFile) Get(label string) ([]string, bool) {
	val, ok := tf.tags[label]
	return val, ok
}

func (tf *TagFile) Len() int {
	return len(tf.labels)
}

func (tf *TagFile) Labels() []string {
	return tf.labels
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
		err := fmt.Errorf("Syntax error at line: %d", lineNum)
		if emptyLineRE.MatchString(line) {
			continue // ignore empty lines
		}
		if contLineRE.MatchString(line) {
			// continuation of previous label
			l := len(tf.labels)
			if l > 0 {
				prevLabel := tf.labels[l-1]
				valIndx := len(tf.tags[prevLabel]) - 1
				tf.tags[prevLabel][valIndx] += " " + strings.Trim(line, ` `)
				continue
			}
			return nil, err
		}
		// must be start of a new label/value pair
		match := labelLineRe.FindStringSubmatch(line)
		if len(match) < 3 {
			return nil, err
		}
		label := strings.Trim(match[1], ` `)
		value := strings.Trim(match[2], ` `)
		tf.Append(label, value)
	}
	return tf, nil
}

func getBagitTxtValues(tf *TagFile) (vers string, enc string, err error) {
	labels := []string{`BagIt-Version`, `Tag-File-Character-Encoding`}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(\d+)\.(\d+)`),
		regexp.MustCompile(`^\S`),
	}
	returnVals := [...]string{``, ``}
	if len(tf.labels) != 2 {
		err = fmt.Errorf(`%s should have %s and %s`, bagitTxt, labels[0], labels[1])
		return ``, ``, err
	}
	for i, label := range tf.labels {
		if label != labels[i] {
			err = fmt.Errorf(`Expected %s in line %d of %s to`, labels[i], i, bagitTxt)
			return ``, ``, err
		}
		vals, ok := tf.tags[label]
		if !ok || len(vals) != 1 {
			err = fmt.Errorf(`Expected 1 entry for %s in %s to`, label, bagitTxt)
			return ``, ``, err
		}
		if !patterns[i].MatchString(vals[0]) {
			err = fmt.Errorf(`Bad value for %s in %s: %s`, label, bagitTxt, vals[0])
			return ``, ``, err
		}
		returnVals[i] = vals[0]
	}
	return returnVals[0], returnVals[1], nil
}
