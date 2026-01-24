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
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, "", fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, "", fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, "", fmt.Errorf("http error: %s", resp.Status)
	}

	return resp.Body, resp.ContentLength, resp.Header.Get("Content-Type"), nil
}
