package bago

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
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
	jobs    chan ChecksumJob
	results chan ChecksumJob
	cancel  chan struct{}
	pushErr chan error
	fs      Backend
}

type ChecksumJob struct {
	Path     string
	Alg      string
	Sum      []byte
	Expected []byte
	Err      error
}

type ChecksumPusher func(ChecksumJob)

func (j *ChecksumJob) SumIsExpected() bool {
	return j.Sum != nil && (bytes.Compare(j.Expected, j.Sum) == 0)
}

func (j *ChecksumJob) SumString() string {
	return hex.EncodeToString(j.Sum)
}

func (j *ChecksumJob) ExpectedString() string {
	return hex.EncodeToString(j.Expected)
}

func NewChecksumer(wkc int, fs Backend, p func(ChecksumPusher) error) *Checksumer {
	c := &Checksumer{
		fs:      fs,
		jobs:    make(chan ChecksumJob),
		results: make(chan ChecksumJob),
		cancel:  make(chan struct{}),
		pushErr: make(chan error, 1),
	}
	var wg sync.WaitGroup
	go func() {
		c.pushErr <- p(func(j ChecksumJob) {
			if c.Canceled() {
				return
			}
			c.jobs <- j
		})
		close(c.jobs)
		close(c.pushErr)
	}()
	for i := 0; i < wkc; i++ {
		wg.Add(1) //checksum workers
		go func() {
			defer wg.Done()
			for job := range c.jobs {
				c.Check(&job)
				c.results <- job
			}
		}()
	}
	go func() {
		wg.Wait()
		close(c.results)
	}()
	return c
}

func (ch *Checksumer) Check(j *ChecksumJob) error {
	if j.Err != nil {
		return j.Err
	}
	var h hash.Hash
	if h, j.Err = NewHash(j.Alg); j.Err != nil {
		return j.Err
	}
	var file io.ReadCloser
	if file, j.Err = ch.fs.Open(j.Path); j.Err != nil {
		return j.Err
	}
	defer file.Close()
	if _, j.Err = io.Copy(h, file); j.Err != nil {
		return j.Err
	}
	j.Sum = h.Sum(nil)
	return nil
}

func (ch *Checksumer) Results() <-chan ChecksumJob {
	return ch.results
}

func (ch *Checksumer) PushError() <-chan error {
	return ch.pushErr
}

func (ch *Checksumer) Cancel() {
	select {
	case <-ch.cancel:
		return // already closed
	default:
		close(ch.cancel)
	}
}

func (ch *Checksumer) Canceled() bool {
	select {
	case <-ch.cancel:
		return true
	default:
		return false
	}
}

func NormalizeAlgName(alg string) (string, error) {
	alg = strings.Replace(alg, `-`, ``, 1)
	alg = strings.ToLower(alg)
	for _, a := range availableAlgs {
		if a == alg {
			return alg, nil
		}
	}
	return ``, fmt.Errorf(`Unknown checksum algorithm: %s`, alg)
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
		return nil, fmt.Errorf(`Unknown checksum algorithm: %s`, alg)
	}
	return h, nil
}
