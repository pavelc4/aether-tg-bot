package utils

import (
	"net/http"
	"time"
)

// HTTP client untuk Telegram Bot API (long polling)
var botClient = &http.Client{
	Timeout: 90 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		DisableKeepAlives:     false,
	},
}

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

// GetBotClient returns HTTP client untuk Telegram Bot API
func GetBotClient() *http.Client {
	return botClient
}

// GetDownloadClient returns HTTP client untuk downloads
func GetDownloadClient() *http.Client {
	return downloadClient
}
