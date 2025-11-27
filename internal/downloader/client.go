package downloader

import (
	"net/http"

	pkghttp "github.com/pavelc4/aether-tg-bot/pkg/http"
)

func GetDownloadClient() *http.Client {
	return pkghttp.GetDownloadClient()
}
