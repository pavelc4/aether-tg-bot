package downloader

import (
	"log"
	"os"
	"path/filepath"
)

// CleanupTempFiles removes temporary directories safely
func CleanupTempFiles(filePaths []string) {
	for _, path := range filePaths {
		if path == "" {
			continue
		}

		dir := filepath.Dir(path)
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("Warning: Failed to cleanup temp directory %s: %v", dir, err)
		}
	}
}

// maxInt returns the maximum of two integers (for Go < 1.21 compatibility)
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
