package utils

import (
	"net/http"

	pkghttp "github.com/pavelc4/aether-tg-bot/pkg/http"
)

func GetBotClient() *http.Client {
	return pkghttp.GetBotClient()
}

func GetDownloadClient() *http.Client {
	return pkghttp.GetDownloadClient()
}
