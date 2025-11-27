package ui

import (
	"fmt"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/internal/downloader/core"
)

func NewUploadTracker(bot *tgbotapi.BotAPI, chatID int64, msgID int, fileName string, totalSize int64, username string) *UploadTracker {
	return &UploadTracker{
		bot:        bot,
		chatID:     chatID,
		msgID:      msgID,
		fileName:   fileName,
		totalSize:  totalSize,
		username:   username,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}
}

func (ut *UploadTracker) Update(uploadedBytes int64) {
	now := time.Now()
	if now.Sub(ut.lastUpdate) < 2*time.Second {
		return
	}

	ut.uploaded = uploadedBytes
	percentage := float64(uploadedBytes) / float64(ut.totalSize) * 100

	elapsed := now.Sub(ut.startTime).Seconds()
	speed := float64(uploadedBytes) / elapsed

	progress := UploadProgress{
		Percentage: percentage,
		Uploaded:   core.FormatBytes(float64(uploadedBytes)),
		TotalSize:  core.FormatBytes(float64(ut.totalSize)),
		Speed:      fmt.Sprintf("%s/s", core.FormatBytes(speed)),
	}

	UpdateUploadProgressMessage(ut.bot, ut.chatID, ut.msgID, ut.fileName, progress, ut.username)
	ut.lastUpdate = now
}

func (ut *UploadTracker) Complete() {
	duration := time.Since(ut.startTime)
	UpdateUploadCompleteMessage(
		ut.bot,
		ut.chatID,
		ut.msgID,
		ut.fileName,
		core.FormatBytes(float64(ut.totalSize)),
		fmt.Sprintf("%.1fs", duration.Seconds()),
		ut.username,
	)
}

type ProgressReader struct {
	file      *os.File
	totalRead int64
	totalSize int64
	tracker   *UploadTracker
}

func NewProgressReader(filePath string, tracker *UploadTracker) (*ProgressReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	return &ProgressReader{
		file:      file,
		totalSize: fileInfo.Size(),
		tracker:   tracker,
	}, nil
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.file.Read(p)
	pr.totalRead += int64(n)

	if pr.tracker != nil {
		pr.tracker.Update(pr.totalRead)
	}

	return n, err
}

func (pr *ProgressReader) Close() error {
	return pr.file.Close()
}
