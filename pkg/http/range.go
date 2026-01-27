package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// Download chunks of 10MB to minimize overhead while bypassing throttling
	chunkSize = 10 * 1024 * 1024 
)

type ChunkedReader struct {
	ctx        context.Context
	url        string
	headers    map[string]string
	client     *http.Client
	offset     int64
	totalSize  int64
	currentBody io.ReadCloser
	err        error
}

func NewChunkedReader(ctx context.Context, url string, headers map[string]string, totalSize int64) *ChunkedReader {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	return &ChunkedReader{
		ctx:       ctx,
		url:       url,
		headers:   headers,
		totalSize: totalSize,
		client:    &http.Client{Transport: transport},
	}
}

func (r *ChunkedReader) Read(p []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}

	if r.currentBody == nil {
		if err := r.nextChunk(); err != nil {
			r.err = err
			return 0, err
		}
	}

	n, err = r.currentBody.Read(p)
	r.offset += int64(n)

	if err == io.EOF {
		r.currentBody.Close()
		r.currentBody = nil
		
		if r.totalSize > 0 && r.offset >= r.totalSize {
			return n, io.EOF
		}
		
		if n > 0 {
			return n, nil 
		}
		
		return r.Read(p)
	}

	return n, err
}

func (r *ChunkedReader) nextChunk() error {
	if r.totalSize > 0 && r.offset >= r.totalSize {
		return io.EOF
	}

	end := r.offset + chunkSize - 1
	if r.totalSize > 0 && end >= r.totalSize {
		end = r.totalSize - 1
	}

	req, err := http.NewRequestWithContext(r.ctx, "GET", r.url, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	// Set Headers
	if _, ok := r.headers["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	// Set Range
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", r.offset, end))

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("range request failed: %w", err)
	}

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	r.currentBody = resp.Body
	return nil
}

func (r *ChunkedReader) Close() error {
	if r.currentBody != nil {
		return r.currentBody.Close()
	}
	return nil
}
