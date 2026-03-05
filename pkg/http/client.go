package http

import (
	"net/http"
	"time"
)

var botClient = &http.Client{
	Timeout: 90 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   50,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		DisableKeepAlives:     false,
		WriteBufferSize:       32 * 1024,
		ReadBufferSize:        32 * 1024,
	},
}

var downloadClient = &http.Client{
	Timeout: 10 * time.Minute,
	Transport: &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       180 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 90 * time.Second,
		DisableKeepAlives:     false,
		WriteBufferSize:       64 * 1024,
		ReadBufferSize:        64 * 1024,
	},
}

func GetBotClient() *http.Client {
	return botClient
}

func GetDownloadClient() *http.Client {
	return downloadClient
}
