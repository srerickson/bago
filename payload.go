package bago

import (
	"fmt"
	"path/filepath"
)

type Payload struct {
	entries map[string]PayloadEntry
}

type PayloadEntry struct {
	rawPath string
	size    int64
}

func initPayload(bagPath string) (*Payload, error) {
	payload := &Payload{}

	fileChan := fileWalker(filepath.Join(bagPath, dataDir))
	for f := range fileChan {
		relPath, _ := filepath.Rel(bagPath, f.path) // file path relative to bag
		encPath := encodePath(relPath)              // encoded for manifests
		// check if already exists?
		if _, exists := payload.entries[encPath]; exists {
			return nil, fmt.Errorf("File path encoding collision with: %s", encPath)
		}
		entry := PayloadEntry{rawPath: f.path, size: f.info.Size()}
		payload.entries[encPath] = entry
	}
	return payload, nil
}
