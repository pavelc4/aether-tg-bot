package handlers

import (
	"os"

	"github.com/pavelc4/aether-tg-bot/internal/downloader/ui"
)

type ProgressReader struct {
	file      *os.File
	totalRead int64
	totalSize int64
	tracker   *ui.UploadTracker
}
