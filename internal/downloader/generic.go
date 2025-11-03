package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func downloadFile(mediaURL, suggestedFilename string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "aether-scrape-")
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := GetDownloadClient().Do(req)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	ext := determineFileExtension(resp, mediaURL, suggestedFilename)
	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))

	file, err := os.Create(filePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	if size < minFileSize {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("file too small: %d bytes", size)
	}

	return filePath, nil
}

func determineFileExtension(resp *http.Response, mediaURL, suggestedFilename string) string {
	if ext := filepath.Ext(suggestedFilename); ext != "" {
		return ext
	}

	contentType := resp.Header.Get("Content-Type")
	for ct, ext := range contentTypeToExt {
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

func calculateTotalSize(filePaths []string, provider string) ([]string, int64, string, error) {
	if len(filePaths) == 0 {
		return nil, 0, "", fmt.Errorf("no files downloaded")
	}

	var totalSize int64
	for _, path := range filePaths {
		if info, err := os.Stat(path); err == nil {
			totalSize += info.Size()
		}
	}

	return filePaths, totalSize, provider, nil
}
