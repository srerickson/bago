package bago

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Payload struct {
	entries map[string]*PayloadEntry
}

type PayloadEntry struct {
	rawPath string
	size    int64
}

func loadPayload(bagPath string) (*Payload, error) {
	payload := &Payload{}
	payload.entries = map[string]*PayloadEntry{}
	payloadPath := filepath.Join(bagPath, dataDir)
	dataInfo, err := os.Stat(payloadPath)
	if err != nil {
		return nil, err
	}
	if !dataInfo.IsDir() {
		return nil, errors.New(`no data directory`)
	}
	fileChan := fileWalker(payloadPath)
	for f := range fileChan {
		relPath, _ := filepath.Rel(bagPath, f.path) // file path relative to bag
		encPath := encodePath(relPath)              // encoded for manifests
		// check if already exists?
		if _, exists := payload.entries[encPath]; exists {
			return nil, fmt.Errorf("File path encoding collision with: %s", encPath)
		}
		entry := &PayloadEntry{rawPath: f.path, size: f.info.Size()}
		payload.entries[encPath] = entry
	}
	return payload, nil
}
