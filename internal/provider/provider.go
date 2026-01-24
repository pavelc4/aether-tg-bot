package provider

import (
	"context"
)

type VideoInfo struct {
	URL      string            // Direct download URL
	FileName string            // Suggested filename
	Title    string            // Title of the media
	Caption  string            // Description/Caption of the media
	FileSize int64             // File size in bytes (0 if unknown)
	MimeType string            // MIME type (video/mp4, etc.)
	Duration int               // Duration in seconds
	Width    int               // Video width
	Height   int               // Video height
	Headers  map[string]string // Required headers for the request (cookies, referer, etc.)
}

type Provider interface {
	Name() string
	Supports(url string) bool
	GetVideoInfo(ctx context.Context, url string) ([]VideoInfo, error)
}
