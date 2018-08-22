package bago

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	defaultVersion = "1.0" // BagIt version for new bags
	bagitTxt       = `bagit.txt`
	dataDir        = `data`
)

// Bag is a bagit repository
type Bag struct {
	path         string
	version      string
	payload      *Payload
	manifests    []*Manifest
	tagManifests []*Manifest
	tagFiles     map[string]*TagFile
	manInPay     map[*ManifestEntry]*Payload
	payInMan     map[*PayloadEntry][]*Manifest
}

// LoadBag returs Bag object for bag at path
func LoadBag(path string) (*Bag, error) {
	bag := &Bag{}
	bag.path = path
	// data payload present?
	dataInfo, err := os.Stat(filepath.Join(path, dataDir))
	if err != nil {
		return nil, err
	}
	if !dataInfo.IsDir() {
		return nil, errors.New(`no data directory`)
	}
	bag.payload, err = initPayload(bag.path)
	if err != nil {
		return nil, err
	}

	// read bagit.txt
	bagitTags, err := ReadTagFile(filepath.Join(path, bagitTxt))
	if err != nil {
		return nil, err
	}
	bag.tagFiles = make(map[string]*TagFile)
	bag.tagFiles[bagitTxt] = bagitTags
	// read manifests for both payload and tag files
	mans, err := ReadAllManifests(path)
	if err != nil {
		return nil, err
	}
	for i := range mans {
		if mans[i].kind == payloadManifest {
			bag.manifests = append(bag.manifests, mans[i])
		} else if mans[i].kind == tagManifest {
			bag.tagManifests = append(bag.tagManifests, mans[i])
		}
	}
	if len(bag.manifests) == 0 {
		return nil, errors.New(`no manifests found`)
	}
	return bag, nil
}

// IsComplete returns whether bag satisfies bag completeness conditions.
// See: https://tools.ietf.org/html/draft-kunze-bagit-16#section-3
func (b *Bag) IsComplete() (bool, error) {
	err := b.bagitTxtErrors()
	if err != nil {
		return false, err
	}
	err = b.statTags()
	if err != nil {
		return false, err
	}
	missingFromManifest := []string{}
	for p, e := range b.payload.entries {
		if len(e.in) == 0 {
			// some versions of spec only require that files are listed in *one* manifest
			missingFromManifest = append(missingFromManifest, p)
		}
	}
	missingFromPayload := []string{}
	for _, m := range b.manifests {
		for p, _ := range m.entries {
			if _, ok := b.payload.entries[p]; !ok {
				missingFromPayload = append(missingFromPayload, p)
			}
		}
	}
	if len(missingFromManifest) > 0 {
		return false, fmt.Errorf("Missing file in manifest: \n --%s", strings.Join(missingFromManifest, ", "))
	}
	if len(missingFromPayload) > 0 {
		return false, fmt.Errorf("Missing file in payload: \n -- %s", strings.Join(missingFromPayload, ",\n -- "))
	}
	return true, nil
}

// IsValid returns whether the bag at path is valid
func (b *Bag) IsValid() (bool, error) {
	_, err := b.IsComplete()
	if err != nil {
		return false, err
	}
	// queue up checksum jobs
	jobInput := make(chan checksumJob)
	jobOutput := checksumWorkers(2, jobInput)
	go func() {
		defer close(jobInput)
		// checksums for all manifest entries
		for _, m := range append(b.manifests, b.tagManifests...) {
			for path, entry := range m.entries {
				jobInput <- checksumJob{
					path:        filepath.Join(b.path, decodePath(path)),
					alg:         m.algorithm,
					expectedSum: entry.sum,
				}
			}
		}
	}()
	errs := []error{}
	for job := range jobOutput {
		if job.expectedSum != job.currentSum {
			errs = append(errs, fmt.Errorf("Checksum failed for: %s", job.path))
		}
	}
	if len(errs) > 0 {
		return false, errs[0]
	}
	return true, nil
}

// Print Bag Contents
func (b *Bag) Print() {
	fmt.Println(b.path)
	for i := range b.manifests {
		fmt.Println(b.manifests[i].algorithm)
		for path, e := range b.manifests[i].entries {
			fmt.Printf("-- %s: %s\n", path, e.sum)
		}
	}
}

func (b *Bag) bagitTxtErrors() error {
	required := map[string]*regexp.Regexp{
		`BagIt-Version`:               regexp.MustCompile(`(\d+)\.(\d+)`),
		`Tag-File-Character-Encoding`: regexp.MustCompile(`(.*)`),
	}
	bagit := b.tagFiles[bagitTxt]
	if bagit == nil {
		return errors.New(`Missing bagit.txt`)
	}
	if bagit.hasBOM {
		msg := fmt.Sprintf(`%s has BOM`, bagitTxt)
		return errors.New(msg)
	}
	for label, pattern := range required {
		if bagit.tags[label] == `` {
			msg := fmt.Sprintf(`Required field missing in %s: %s`, bagitTxt, label)
			return errors.New(msg)
		}
		if !pattern.MatchString(bagit.tags[label]) {
			msg := fmt.Sprintf(`Malformed value in %s for label %s: %s`, bagitTxt, label, bagit.tags[label])
			return errors.New(msg)
		}
	}
	return nil
}

// TODO return all failed
func (b *Bag) statTags() error {
	for _, m := range b.tagManifests {
		for _, e := range m.entries {
			_, err := os.Stat(filepath.Join(b.path, e.rawPath))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
