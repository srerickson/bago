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

type Checksumer struct {
	workers int
	jobs    chan checksumJob
	results chan checksumJob
	cancel  chan error
	err     error // error that caused cancel
	fs      Backend
}

type JobPushFunc func(string, string, string)

type checksumJob struct {
	path        string
	alg         string
	expectedSum string
	actualSum   string
	// err         error
}

func NewChecksumer(workers int, backend Backend) *Checksumer {
	c := &Checksumer{workers: workers, fs: backend}
	c.jobs = make(chan checksumJob)
	c.results = make(chan checksumJob)
	c.cancel = make(chan error)

	go func() {
		for {
			select {
			case c.err = <-c.cancel:
				for _ = range c.jobs {
				}
			}
		}
		close(c.cancel)
	}()
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1) //checksum workers
		go func() {
			defer wg.Done()
			for job := range c.jobs {
				var err error
				if job.actualSum, err = c.Check(job); err != nil {
					c.cancel <- err
					break
				}
				c.results <- job
			}
		}()
	}
	go func() {
		wg.Wait() // for workers to complete
		close(c.results)
	}()
	return c
}

func (ch *Checksumer) Check(j checksumJob) (string, error) {
	h, err := NewHash(j.alg)
	if err != nil {
		return "", err
	}
	file, err := ch.fs.Open(j.path)
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

func (ch *Checksumer) Push(p string, a string, s string) {
	ch.jobs <- checksumJob{
		path:        p,
		alg:         a,
		expectedSum: s,
	}
}

func (ch *Checksumer) PushFunc(f func(JobPushFunc) error) {
	go func() {
		defer close(ch.jobs)
		if err := f(ch.Push); err != nil {
			ch.cancel <- err
		}
	}()
}

func (ch *Checksumer) Results() <-chan checksumJob {
	return ch.results
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
