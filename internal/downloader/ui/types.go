package ui

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	barLength  = 10
	filledChar = "■"
	emptyChar  = "□"
)

type DownloadProgress struct {
	Percentage float64
	Downloaded string
	Speed      string
	ETA        string
	Status     string
}
type UploadTracker struct {
	bot        *tgbotapi.BotAPI
	chatID     int64
	msgID      int
	fileName   string
	totalSize  int64
	username   string
	startTime  time.Time
	lastUpdate time.Time
	uploaded   int64
}

type UploadProgress struct {
	Percentage float64
	Uploaded   string
	TotalSize  string
	Speed      string
}
