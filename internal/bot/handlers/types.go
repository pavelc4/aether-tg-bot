package handlers

import (
	"os"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/pavelc4/aether-tg-bot/internal/downloader/ui"
)

var extGroups = map[string]string{
	".jpg": "photos", ".jpeg": "photos", ".png": "photos", ".webp": "photos",
	".mp4": "videos", ".webm": "videos", ".mkv": "videos",
	".mp3": "audios", ".m4a": "audios", ".ogg": "audios", ".flac": "audios", ".wav": "audios", ".opus": "audios",
}

type Handler struct {
	Client *telegram.Client
	Sender *message.Sender
}

type fileGroup struct {
	name  string
	files []string
}

type ProgressReader struct {
	file      *os.File
	totalRead int64
	totalSize int64
	tracker   *ui.UploadTracker
}
