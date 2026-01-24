package provider

import (
	"context"
)

type VideoInfo struct {
	URL      string            // Direct download URL
	FileName string            // Suggested filename
	FileSize int64             // File size in bytes (0 if unknown)
	MimeType string            // MIME type (video/mp4, etc.)
	Duration int               // Duration in seconds
	Headers  map[string]string // Required headers for the request (cookies, referer, etc.)
}

type Provider interface {
	Name() string
	Supports(url string) bool
	GetVideoInfo(ctx context.Context, url string) ([]VideoInfo, error)
}
