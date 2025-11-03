package utils

import (
	"context"
	"log"
	"os"
	"path/filepath"
)

var TempFilePatterns = []string{
	"aether-ytdlp-*",
	"aether-scrape-*",
	"aether-image-*",
	"aether-tiktok-audio-*",
}

func CleanupTempFiles(filePaths []string) {
	for _, path := range filePaths {
		if path == "" {
			continue
		}

		dir := filepath.Dir(path)
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("  Failed to cleanup temp directory %s: %v", dir, err)
		}
	}
}

func CleanupTempFilesByPattern(ctx context.Context, patterns []string) int {
	log.Println("🧹 Cleaning up temporary files...")

	tempDir := os.TempDir()
	filesCleaned := 0

	for _, pattern := range patterns {
		select {
		case <-ctx.Done():
			log.Printf("  Cleanup cancelled due to timeout (%d files cleaned)", filesCleaned)
			return filesCleaned
		default:
		}

		fullPattern := filepath.Join(tempDir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			log.Printf("  Error finding temp files with pattern %s: %v", pattern, err)
			continue
		}

		for _, path := range matches {
			select {
			case <-ctx.Done():
				log.Printf("  Cleanup cancelled during file operations (%d files cleaned)", filesCleaned)
				return filesCleaned
			default:
			}

			if err := os.RemoveAll(path); err != nil {
				log.Printf("  Failed to remove %s: %v", path, err)
			} else {
				filesCleaned++
				log.Printf(" Cleaned up: %s", filepath.Base(path))
			}
		}
	}

	log.Printf(" Temp files cleanup completed. %d files/directories cleaned.", filesCleaned)
	return filesCleaned
}

func DeleteDirectory(path string) error {
	if err := os.RemoveAll(path); err != nil {
		log.Printf("  Failed to delete directory %s: %v", path, err)
		return err
	}
	return nil
}
