package downloader

import (
	"log"
	"os"
	"path/filepath"
)

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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
