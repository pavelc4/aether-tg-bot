package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	pkghttp "github.com/pavelc4/aether-tg-bot/pkg/http"
)

func NewFileDownloader(timeout time.Duration) *FileDownloader {
	return &FileDownloader{
		timeout: timeout,
		client:  pkghttp.GetDownloadClient(),
	}
}

func (fd *FileDownloader) DownloadFile(ctx context.Context, mediaURL, suggestedFilename string) (string, int64, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), fd.timeout)
		defer cancel()
	}

	tmpDir, err := os.MkdirTemp("", "aether-download-")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp directory: %w", err)
	}

	filePath, size, err := fd.downloadAndSave(ctx, mediaURL, suggestedFilename, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}

	return filePath, size, nil
}

func (fd *FileDownloader) downloadAndSave(ctx context.Context, mediaURL, suggestedFilename, tmpDir string) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := fd.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	ext := fd.DetermineFileExtension(resp, mediaURL, suggestedFilename)
	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))

	file, err := os.Create(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return "", 0, err
	}

	if size < MinFileSize {
		return "", 0, fmt.Errorf("file too small: %d bytes", size)
	}

	return filePath, size, nil
}

func (fd *FileDownloader) DetermineFileExtension(resp *http.Response, mediaURL, suggestedFilename string) string {
	if ext := filepath.Ext(suggestedFilename); ext != "" {
		return ext
	}

	contentType := resp.Header.Get("Content-Type")
	for ct, ext := range ContentTypeToExt {
		if strings.Contains(contentType, ct) {
			return ext
		}
	}

	if parsedURL, err := url.Parse(mediaURL); err == nil {
		if ext := filepath.Ext(parsedURL.Path); ext != "" {
			return ext
		}
	}

	return ".tmp"
}

func CalculateTotalSize(filePaths []string) (int64, error) {
	if len(filePaths) == 0 {
		return 0, fmt.Errorf("no files provided")
	}

	var totalSize int64
	for _, path := range filePaths {
		if info, err := os.Stat(path); err == nil {
			totalSize += info.Size()
		}
	}

	return totalSize, nil
}

func CleanupFiles(filePaths []string) {
	if len(filePaths) == 0 {
		return
	}

	dirs := make(map[string]bool)
	for _, path := range filePaths {
		dir := filepath.Dir(path)
		dirs[dir] = true
	}

	for dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("Warning: Failed to cleanup temp directory %s: %v", dir, err)
		}
	}
}

func ValidateFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is directory, not file")
	}

	if info.Size() < MinFileSize {
		return fmt.Errorf("file too small: %d bytes", info.Size())
	}

	return nil
}
