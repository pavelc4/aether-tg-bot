package downloader

import (
	httpclient "github.com/pavelc4/aether-tg-bot/pkg/http"
	"net/http"
)

func GetDownloadClient() *http.Client {
	return httpclient.GetDownloadClient()
}
