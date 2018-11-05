package test

import (
	"path/filepath"
	"runtime"
)

// Returns absolute path to test data
func DataPath(relPath []string) string {
	_, fPath, _, _ := runtime.Caller(0)
	absPath := make([]string, len(relPath)+1)
	absPath[0] = filepath.Dir(fPath)
	for i := range relPath {
		absPath[i+1] = relPath[i]
	}
	return filepath.Join(absPath...)
}
