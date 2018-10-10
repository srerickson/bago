package bago

import (
	"strings"
	"testing"
)

func TestManifestParse(t *testing.T) {
	tests := map[bool][]string{
		true: []string{ // valid
			"1234 file1\n5678 file2",
			"9ABC\tfile3",
			"DEFG afile%0Awith%0Dspecial%25characters\nHIJKL another_file",
		},
		false: []string{ // invalid
			``,
			"\n123 afile",
			`123`,
			` 1234 afile`,
			"123 file1\n123 file1",
			"123 file1\n567 file1",
		},
	}
	for expectValid, vals := range tests {
		for i := range vals {
			r := strings.NewReader(vals[i])
			m := &Manifest{}
			if err := m.parse(r); (err == nil) != expectValid {
				if expectValid {
					t.Errorf("expected parse to return nil for `%s`: error is %s", vals[i], err)
				} else {
					t.Errorf("expected parse to return error for `%s`", vals[i])
				}
			}
		}
	}

}
