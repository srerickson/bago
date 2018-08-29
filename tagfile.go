package bago

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type TagSet map[string][]string

type TagFile struct {
	tags   TagSet
	labels []string
}

type bagitValues struct {
	encoding string
	version  [2]int
}

func (tf *TagFile) append(label string, value string) []string {
	if tf.tags == nil {
		tf.tags = TagSet{}
	}
	if tf.labels == nil {
		tf.labels = []string{}
	}
	if _, ok := tf.tags[label]; !ok {
		tf.tags[label] = []string{value}
		tf.labels = append(tf.labels, label)
	} else {
		tf.tags[label] = append(tf.tags[label], value)
	}
	return tf.tags[label]
}

func (tf *TagFile) parse(reader io.Reader) error {
	tf.tags = nil
	tf.labels = nil
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
			return err
		}
		// must be start of a new label/value pair
		match := labelLineRe.FindStringSubmatch(line)
		if len(match) < 3 {
			return err
		}
		label := strings.Trim(match[1], ` `)
		value := strings.Trim(match[2], ` `)
		tf.append(label, value)
	}
	return nil
}

// bagitTxtValues validates structure of TagFile from bagit.txt and returns
// tag values.
func (tf *TagFile) bagitTxtValues() (ret bagitValues, err error) {
	labels := [...]string{`BagIt-Version`, `Tag-File-Character-Encoding`}
	patterns := [...]*regexp.Regexp{
		regexp.MustCompile(`(\d+)\.(\d+)`),
		regexp.MustCompile(`^(\S+)`),
	}
	tmpVals := []string{}
	if len(tf.labels) != len(labels) {
		err = fmt.Errorf(`%s should have %s and %s`, bagitTxt, labels[0], labels[1])
		return ret, err
	}
	for i, label := range tf.labels {
		if label != labels[i] {
			err = fmt.Errorf(`Expected %s in line %d of %s to`, label, i, bagitTxt)
			return ret, err
		}
		vals, ok := tf.tags[label]
		if !ok || len(vals) != 1 {
			err = fmt.Errorf(`Expected 1 entry for %s in %s to`, label, bagitTxt)
			return ret, err
		}
		matches := patterns[i].FindStringSubmatch(vals[0])
		if len(matches) == 0 {
			err = fmt.Errorf(`Bad value for %s in %s: %s`, label, bagitTxt, vals[0])
			return ret, err
		}
		tmpVals = append(tmpVals, matches[1:]...)
	}
	if len(tmpVals) != 3 {
		return ret, fmt.Errorf(`unexpected values parsing %s`, bagitTxt)
	}
	ret.version[0], _ = strconv.Atoi(tmpVals[0])
	ret.version[1], _ = strconv.Atoi(tmpVals[1])
	ret.encoding = tmpVals[2]
	return ret, nil
}
