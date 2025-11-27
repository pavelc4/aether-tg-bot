package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/internal/downloader/ui"
)

func NewProgressReader(filePath string, tracker *ui.UploadTracker) (*ProgressReader, error) {
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

func sendText(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send text: %v", err)
	}
}

func deleteMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int) {
	del := tgbotapi.NewDeleteMessage(chatID, msgID)
	if _, err := bot.Request(del); err != nil {
		log.Printf("Failed to delete message: %v", err)
	}
}

// SendMediaOptions configuration for sending media
type SendMediaOptions struct {
	Caption  string
	MsgID    int
	Username string
	IsVideo  bool
}

// SendMedia unified function to send media with progress
func SendMedia(bot *tgbotapi.BotAPI, chatID int64, filePath string, opts SendMediaOptions) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Failed to stat file: %v", err)
		sendSimpleMedia(bot, chatID, filePath, opts)
		return
	}

	fileName := filepath.Base(filePath)
	fileSize := fileInfo.Size()

	var tracker *ui.UploadTracker
	if opts.MsgID > 0 && opts.Username != "" {
		tracker = ui.NewUploadTracker(bot, chatID, opts.MsgID, fileName, fileSize, opts.Username)
		log.Printf("Starting upload: %s (%.2f MB)", fileName, float64(fileSize)/(1024*1024))
	}

	progressReader, err := NewProgressReader(filePath, tracker)
	if err != nil {
		log.Printf("Failed to open file for progress: %v", err)
		sendSimpleMedia(bot, chatID, filePath, opts)
		return
	}
	defer progressReader.Close()

	var msg tgbotapi.Chattable
	fileReader := tgbotapi.FileReader{Name: fileName, Reader: progressReader}

	if opts.IsVideo {
		v := tgbotapi.NewVideo(chatID, fileReader)
		v.SupportsStreaming = true
		if opts.Caption != "" {
			v.Caption = opts.Caption
			v.ParseMode = "Markdown"
		}
		msg = v
	} else {
		a := tgbotapi.NewAudio(chatID, fileReader)
		if opts.Caption != "" {
			a.Caption = opts.Caption
			a.ParseMode = "Markdown"
		}
		msg = a
	}

	startTime := time.Now()
	if _, err = bot.Send(msg); err != nil {
		log.Printf("Failed to send media: %v", err)
		return
	}

	if tracker != nil {
		tracker.Complete()
		duration := time.Since(startTime)
		log.Printf("Upload complete: %s in %.1fs (%.2f MB/s)",
			fileName,
			duration.Seconds(),
			float64(fileSize)/(1024*1024)/duration.Seconds(),
		)
	}
}

// sendSimpleMedia fallback function without progress tracking
func sendSimpleMedia(bot *tgbotapi.BotAPI, chatID int64, filePath string, opts SendMediaOptions) {
	var msg tgbotapi.Chattable
	if opts.IsVideo {
		v := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
		v.SupportsStreaming = true
		if opts.Caption != "" {
			v.Caption = opts.Caption
			v.ParseMode = "Markdown"
		}
		msg = v
	} else {
		a := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(filePath))
		if opts.Caption != "" {
			a.Caption = opts.Caption
			a.ParseMode = "Markdown"
		}
		msg = a
	}
	bot.Send(msg)
}

func sendMediaGroup(bot *tgbotapi.BotAPI, chatID int64, filePaths []string, caption string, isVideo bool) error {
	return sendMediaGroupWithProgress(bot, chatID, filePaths, caption, isVideo, 0, "")
}

func sendMediaGroupWithProgress(bot *tgbotapi.BotAPI, chatID int64, filePaths []string, caption string, isVideo bool, msgID int, username string) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("no files to send")
	}

	opts := SendMediaOptions{
		Caption:  caption,
		MsgID:    msgID,
		Username: username,
		IsVideo:  isVideo,
	}

	if len(filePaths) == 1 {
		SendMedia(bot, chatID, filePaths[0], opts)
		return nil
	}

	maxMediaGroupSize := 10
	for i := 0; i < len(filePaths); i += maxMediaGroupSize {
		end := i + maxMediaGroupSize
		if end > len(filePaths) {
			end = len(filePaths)
		}

		batch := filePaths[i:end]
		var mediaGroup []interface{}

		for j, path := range batch {
			var media interface{}
			if isVideo {
				media = tgbotapi.NewInputMediaVideo(tgbotapi.FilePath(path))
			} else {
				media = tgbotapi.NewInputMediaAudio(tgbotapi.FilePath(path))
			}

			if i == 0 && j == 0 {
				// Only first item gets caption
				switch m := media.(type) {
				case tgbotapi.InputMediaVideo:
					m.Caption = caption
					m.ParseMode = "Markdown"
					media = m
				case tgbotapi.InputMediaAudio:
					m.Caption = caption
					m.ParseMode = "Markdown"
					media = m
				}
			}
			mediaGroup = append(mediaGroup, media)
		}

		mediaGroupConfig := tgbotapi.NewMediaGroup(chatID, mediaGroup)
		if _, err := bot.SendMediaGroup(mediaGroupConfig); err != nil {
			log.Printf("Failed to send media group (batch %d-%d): %v", i, end, err)
			// Fallback: send individually
			for k, path := range batch {
				batchOpts := opts
				if i != 0 || k != 0 {
					batchOpts.Caption = "" // Only first file of first batch gets caption
				}
				SendMedia(bot, chatID, path, batchOpts)
			}
		} else {
			log.Printf(" Sent media group: %d files (batch %d-%d)", len(batch), i, end)
		}
	}
	return nil
}
