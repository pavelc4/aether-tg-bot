package providers

import (
	"context"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Provider interface {
	Name() string
	CanHandle(url string) bool
	Download(ctx context.Context, url string, audioOnly bool) ([]string, error)
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
	bot        *tgbotapi.BotAPI
	chatID     int64
	msgID      int
	username   string
	fileName   string
	totalSize  string
}

type TikWMResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		MusicInfo struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Author  string `json:"author"`
			Play    string `json:"play"`
			PlayURL string `json:"play_url"`
		} `json:"music_info"`
		VideoInfo struct {
			DownloadAddr string `json:"downloadAddr"`
			PlayAddr     string `json:"playAddr"`
		} `json:"video"`
		Videos []struct {
			DownloadAddr string `json:"downloadAddr"`
			PlayAddr     string `json:"playAddr"`
		} `json:"videos"`
	} `json:"data"`
}
