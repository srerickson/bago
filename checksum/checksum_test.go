package checksum

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/srerickson/bago/backend"
)

func testDataPath() string {
	_, fPath, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(fPath), `../test/bags`)
}

func testBag() backend.Backend {
	test_path := filepath.Join(testDataPath(), "v0.97", "valid", "bag-in-a-bag")
	return &backend.FS{Path: test_path}
}

func TestChecksumBasic(t *testing.T) {
	for n := 1; n < 4; n++ {
		results := []string{}
		c := New(n, testBag(), func(push JobPusher) error {
			push(Job{Path: `bagit.txt`, Alg: MD5})
			push(Job{Path: `bag-info.txt`, Alg: SHA1})
			push(Job{Path: `manifest-md5.txt`, Alg: SHA256})
			push(Job{Path: `tagmanifest-md5.txt`, Alg: SHA512})
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
	c := New(1, testBag(), func(push JobPusher) error {
		push(Job{Path: `bagit.txt`, Alg: MD5})
		push(Job{Path: `bag-info.txt`, Alg: MD5})
		push(Job{Path: `manifest-md5.txt`, Alg: MD5})
		push(Job{Path: `tagmanifest-md5.txt`, Alg: MD5})
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
	c := New(1, testBag(), func(push JobPusher) error {
		push(Job{Path: `bagit.txt`, Alg: MD5})
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
