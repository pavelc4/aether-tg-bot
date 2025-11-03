package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func DownloadImage(imageURL string) (string, int64, error) {
	tmpDir, err := os.MkdirTemp("", "aether-image-")
	if err != nil {
		return "", 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cobaltTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}

	resp, err := GetDownloadClient().Do(req)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	ext := getImageExtension(resp.Header.Get("Content-Type"))
	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))

	file, err := os.Create(filePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}

	if size < minFileSize {
		os.RemoveAll(tmpDir)
		return "", 0, fmt.Errorf("image too small: %d bytes", size)
	}

	return filePath, size, nil
}

func getImageExtension(contentType string) string {
	for ct, ext := range imageContentTypes {
		if strings.Contains(contentType, ct) {
			return ext
		}
	}
	return ".jpg"
}
