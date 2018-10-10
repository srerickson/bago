package bago

import (
	"strings"
	"testing"
)

func TestTagfileParse(t *testing.T) {
	tests := map[bool][]string{
		true: []string{ // valid
			"Field1: Val1\nField1: Val2",
			"Field1: Val1\n\nField1: Val2",
			"Field1: Val\n On\n  Several\n  Lines\nField2: Val2",
		},
		false: []string{ // invalid
			"Field1",
			"Field1 Val1",
			" Field1: Val1",
		},
	}
	for expectValid, vals := range tests {
		for i := range vals {
			r := strings.NewReader(vals[i])
			tf := &TagFile{}
			if err := tf.parse(r); (err == nil) != expectValid {
				if expectValid {
					t.Errorf("expected TagFile.parse to return nil for `%s`: error is %s", vals[i], err)
				} else {
					t.Errorf("expected TagFile.parse to return error for `%s`", vals[i])
				}
			}
		}
	}

}
