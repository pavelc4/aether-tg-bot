package clinet

import (
	"net/http"
	"time"
)

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

func GetBotClient() *http.Client {
	return botClient
}

// GetDownloadClient returns HTTP client untuk downloads
func GetDownloadClient() *http.Client {
	return downloadClient
}
