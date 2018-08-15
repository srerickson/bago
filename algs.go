package bago

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
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

// NewHash returns Hash object for specified algorithm
func newHash(alg string) (hash.Hash, error) {
	var h hash.Hash
	switch alg {
	case `sha512`:
		h = sha512.New()
	case `sha256`:
		h = sha256.New()
	case `sha1`:
		h = sha1.New()
	case `md5`:
		h = md5.New()
	default:
		return nil, errors.New(`Hash algorithm not specified, or not available`)
	}
	return h, nil
}

// Checksum returns checksum for file with given path using given algorithm
func Checksum(path string, alg string) (string, error) {
	h, err := newHash(alg)
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = io.Copy(h, file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ChecksumConcurrent(path string, alg string, resultC chan string, errorC chan error) {
	sum, err := Checksum(path, alg)
	resultC <- sum
	errorC <- err
}
