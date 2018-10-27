package bago

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
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

type checksumer func(string, string) (string, error)

type checksumJob struct {
	path        string
	alg         string
	checker     checksumer
	expectedSum string
	currentSum  string
	err         error
}

func algIsAvailabe(alg string) bool {
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
	if algIsAvailabe(alg) {
		return alg, nil
	}
	msg := fmt.Sprintf(`Unknown checksum algorithm: %s`, alg)
	return ``, errors.New(msg)
}

// NewHash returns Hash object for specified algorithm
func NewHash(alg string) (hash.Hash, error) {
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

func checksumPool(n int, jobs <-chan checksumJob) <-chan checksumJob {
	results := make(chan checksumJob)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1) //checksum workers
		go func() {
			defer wg.Done()
			for job := range jobs {
				if job.err != nil {
					results <- job
					break
				}
				job.currentSum, job.err = job.checker(job.path, job.alg)
				results <- job
				if job.err != nil {
					break
				}
			}
		}()
	}
	go func() {
		wg.Wait() // for workers to complete
		close(results)
	}()
	return results
}

func (b *Bag) ValidateManifests(workers int) (err error) {
	inQ := make(chan checksumJob)
	outQ := checksumPool(workers, inQ)
	go func() {
		defer close(inQ)
		for _, m := range append(b.manifests, b.tagManifests...) {
			for path, entry := range m.entries {
				inQ <- checksumJob{
					path:        decodePath(path),
					alg:         m.algorithm,
					expectedSum: entry.sum,
					checker:     b.Backend.Checksum,
				}
			}
		}
	}()
	for job := range outQ {
		if job.expectedSum != job.currentSum {
			if err == nil {
				err = errors.New("checksum failed for: ")
			}
			err = fmt.Errorf("%s '%s'", err.Error(), job.path)
		}
	}
	return err
}
