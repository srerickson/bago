package bago

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var algorithmRE = regexp.MustCompile(`.*manifest-(\w+).txt$`)

var algs = [4]string{`sha512`, `sha256`, `sha1`, `md5`}

// ManifestAglorithm returns checksum algorithm from manifest's filename
func ManifestAglorithm(filename string) (string, error) {
	match := algorithmRE.FindStringSubmatch(filename)
	if len(match) == 0 {
		return "", errors.New("Could not determine manifest's checksum algorithm")
	}
	alg := strings.ToLower(match[1])
	for _, a := range algs {
		if a == alg {
			return alg, nil
		}
	}
	msg := fmt.Sprintf("%s is not a recognized checksum algorithm", alg)
	return alg, errors.New(msg)
}
