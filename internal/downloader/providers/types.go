package providers

import (
	"context"
	"net/http"
	"time"
)

type Provider interface {
	Name() string
	CanHandle(url string) bool
	Download(ctx context.Context, url string, audioOnly bool) ([]string, string, error)
}

type DownloadOptions struct {
	URL        string
	AudioOnly  bool
	UseCookies bool
}

type CobaltProvider struct {
	timeout  time.Duration
	client   *http.Client
	handlers map[string]responseHandler
}

type TikTokProvider struct {
	timeout time.Duration
	client  *http.Client
}

type cobaltAPIResponse struct {
	Status   string `json:"status"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Picker   []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"picker"`
	Error struct {
		Code    string      `json:"code"`
		Context interface{} `json:"context"`
	} `json:"error"`
}

type YouTubeProvider struct {
	timeout    time.Duration
	useCookies bool
}

type TikWMResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Play      string   `json:"play"`
		WmPlay    string   `json:"wmplay"`
		Music     string   `json:"music"`
		Title     string   `json:"title"`
		Images    []string `json:"images"`
		MusicInfo struct {
			Title  string `json:"title"`
			Author string `json:"author"`
			Play   string `json:"play"`
		} `json:"music_info"`
	} `json:"data"`
}
