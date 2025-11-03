package utils

import (
	httpclient "github.com/pavelc4/aether-tg-bot/pkg/http"
	"net/http"
)

func GetBotClient() *http.Client {
	return httpclient.GetBotClient()
}

func GetDownloadClient() *http.Client {
	return httpclient.GetDownloadClient()
}
