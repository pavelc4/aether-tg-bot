package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultTimeout = 30 * time.Second
)


func StreamRequest(ctx context.Context, url string, headers map[string]string) (io.ReadCloser, int64, string, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return nil, 0, "", fmt.Errorf("create head request failed: %w", err)
	}

	if _, ok := headers["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, "", fmt.Errorf("head request failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, "", fmt.Errorf("head http error: %s", resp.Status)
	}

	size := resp.ContentLength
	contentType := resp.Header.Get("Content-Type")

	reader := NewChunkedReader(ctx, url, headers, size)
	return reader, size, contentType, nil
}
