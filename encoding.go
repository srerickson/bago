package bago

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/unicode/norm"
)

type EncPath string  // A path encoded for manifest (percent encoded)
type NormPath string // A path encoded and normalized (Unicode-NFC)

// EncodePath encodes a path for inclusion in a manifest file
func EncodePath(s string) EncPath {
	// FIXME: BagIt v. 1.0 requires '%' to be encoded
	// s = strings.Replace(s, `%`, `%25`, -1)
	s = strings.Replace(s, "\r", `%0D`, -1)
	s = strings.Replace(s, "\n", `%0A`, -1)
	s = filepath.ToSlash(s)
	return EncPath(s)
}

func (p EncPath) Decode() string {
	lf := regexp.MustCompile(`(%0[Aa])`)
	cr := regexp.MustCompile(`(%0[Dd])`)
	s := filepath.FromSlash(string(p))
	s = lf.ReplaceAllString(s, "\n")
	s = cr.ReplaceAllString(s, "\r")
	// FIXME
	// s = strings.Replace(s, `%25`, `%`, -1)
	return s
}

func (s EncPath) Norm() NormPath {
	return NormPath(norm.NFC.String(string(s)))
}

func newDecodeReader(reader io.Reader, enc string) (io.Reader, error) {
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
