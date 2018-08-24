package bago

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

func newReader(reader io.Reader, enc string) (io.Reader, error) {
	switch strings.ToLower(enc) {
	case `utf-8`:
		return reader, nil
	case `utf-16`:
		dec := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
		return dec.Reader(reader), nil
	case `iso-8859-1`:
		dec := charmap.ISO8859_1.NewDecoder()
		return dec.Reader(reader), nil
	}
	return nil, fmt.Errorf("Unrecognized encoding: %s", enc)
}
