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
	"strings"
	"sync"
)

// SHA512 = `sha512`
const (
	SHA512 = `sha512`
	SHA256 = `sha256`
	SHA224 = `sha224`
	SHA1   = `sha1`
	MD5    = `md5`
)

var availableAlgs = [...]string{SHA512, SHA256, SHA224, SHA1, MD5}

type checksumJob struct {
	path        string
	alg         string
	expectedSum string
	currentSum  string
	err         error
}

func AlgIsAvailabe(alg string) bool {
	for _, a := range availableAlgs {
		if a == alg {
			return true
		}
	}
	return false
}

func NormalizeAlgName(alg string) (string, error) {
	alg = strings.Replace(alg, `-`, ``, 1)
	alg = strings.ToLower(alg)
	if AlgIsAvailabe(alg) {
		return alg, nil
	}
	msg := fmt.Sprintf(`Uknown checksum algorithm: %s`, alg)
	return ``, errors.New(msg)
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

// NewHash returns Hash object for specified algorithm
func newHash(alg string) (hash.Hash, error) {
	var h hash.Hash
	switch alg {
	case SHA512:
		h = sha512.New()
	case SHA256:
		h = sha256.New()
	case SHA224:
		h = sha256.New224()
	case SHA1:
		h = sha1.New()
	case MD5:
		h = md5.New()
	default:
		msg := fmt.Sprintf(`Hash algorithm not available or not specified: %s`, alg)
		return nil, errors.New(msg)
	}
	return h, nil
}

func checksumWorkers(workers int, jobs chan checksumJob) chan checksumJob {
	results := make(chan checksumJob)
	var wg sync.WaitGroup

	//checksum workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if job.err != nil {
					results <- job
					break
				}
				job.currentSum, job.err = Checksum(job.path, job.alg)
				results <- job
				if job.err != nil {
					break
				}
			}
		}()
	}
	// Channel Closers
	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}
