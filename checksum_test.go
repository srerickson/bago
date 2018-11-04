package bago

import (
	"errors"
	"path/filepath"
	"testing"
)

func testBag() Backend {
	test_path := filepath.Join(testDataPath(), "v0.97", "valid", "bag-in-a-bag")
	return &FSBag{path: test_path}
}

func TestChecksumBasic(t *testing.T) {
	for n := 1; n < 4; n++ {
		results := []string{}
		c := NewChecksumer(n, testBag(), func(push ChecksumPusher) error {
			push(ChecksumJob{Path: `bagit.txt`, Alg: MD5})
			push(ChecksumJob{Path: `bag-info.txt`, Alg: SHA1})
			push(ChecksumJob{Path: `manifest-md5.txt`, Alg: SHA256})
			push(ChecksumJob{Path: `tagmanifest-md5.txt`, Alg: SHA512})
			return nil
		})
		for r := range c.Results() {
			if r.Err != nil {
				t.Errorf("unexpected error: %s", r.Err.Error())
			}
			results = append(results, r.SumString())
		}
		if len(results) != 4 {
			t.Errorf("expected there to be 2 checksum result, not %v", len(results))
		}
	}
}

func TestChecksumCancel(t *testing.T) {
	results := []string{}
	c := NewChecksumer(1, testBag(), func(push ChecksumPusher) error {
		push(ChecksumJob{Path: `bagit.txt`, Alg: MD5})
		push(ChecksumJob{Path: `bag-info.txt`, Alg: MD5})
		push(ChecksumJob{Path: `manifest-md5.txt`, Alg: MD5})
		push(ChecksumJob{Path: `tagmanifest-md5.txt`, Alg: MD5})
		return nil
	})
	go func() {
		c.Cancel()
	}()
	for r := range c.Results() {
		if r.Err != nil {
			t.Errorf("unexpected error: %s", r.Err.Error())
		}
		results = append(results, r.SumString())
	}
	if len(results) >= 4 {
		t.Errorf("expected fewer than four results, not %v", len(results))
	}
}

func TestChecksumPushError(t *testing.T) {
	results := []string{}
	c := NewChecksumer(1, testBag(), func(push ChecksumPusher) error {
		push(ChecksumJob{Path: `bagit.txt`, Alg: MD5})
		return errors.New("a problem")
	})
	for r := range c.Results() {
		if r.Err != nil {
			t.Errorf("unexpected error: %s", r.Err.Error())
		}
		results = append(results, r.SumString())
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, not %v", len(results))
	}
	if e := <-c.PushError(); e == nil || e.Error() != `a problem` {
		t.Error("expected to receive an error, not nil")
	}

}
