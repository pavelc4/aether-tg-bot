package downloader

import (
	"net/http"
	"time"
)

// HTTP client untuk download (Cobalt, yt-dlp, images)
var downloadClient = &http.Client{
	Timeout: 10 * time.Minute,
	Transport: &http.Transport{
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       30,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 90 * time.Second,
		DisableKeepAlives:     false,
	},
}

// GetDownloadClient returns HTTP client untuk downloads
func GetDownloadClient() *http.Client {
	return downloadClient
}
