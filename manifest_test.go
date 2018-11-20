package bago

import (
	"strings"
	"testing"
)

func TestManifestAppend(t *testing.T) {
	m := Manifest{}
	p := EncPath("afile%0Awith%0Dspecial characters")
	m.Append(p, nil)
	entry, exists := m.entries[p.Norm()]
	if !exists {
		t.Error(`Append failed`)
	}
	if entry.path != "afile\nwith\rspecial characters" {
		t.Errorf(`Appended path not decoded correctly. Got: %s`, entry.path)
	}
}

func TestManifestParse(t *testing.T) {
	tests := map[bool][]string{
		true: []string{ // valid
			"1234 file1\n5678 file2",
			"9ABC\tfile3",
			"DEF8 afile%0Awith%0Dspecial%25characters\nABC9 another_file",
		},
		false: []string{ // invalid
			``,
			"\n1234 afile",
			`1234`,
			` 1234 afile`,
			"1234 file1\n123 file1",
			"1234 file1\n567 file1",
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
