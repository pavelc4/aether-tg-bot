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

func sendVideo(bot *tgbotapi.BotAPI, chatID int64, filePath string) {
	sendVideoWithProgress(bot, chatID, filePath, "", 0, "")
}

func sendAudio(bot *tgbotapi.BotAPI, chatID int64, filePath string) {
	sendAudioWithProgress(bot, chatID, filePath, "", 0, "")
}

func sendVideoWithProgress(bot *tgbotapi.BotAPI, chatID int64, filePath, caption string, msgID int, username string) {
	sendMediaWithProgress(bot, chatID, filePath, caption, msgID, username, true)
}

func sendAudioWithProgress(bot *tgbotapi.BotAPI, chatID int64, filePath, caption string, msgID int, username string) {
	sendMediaWithProgress(bot, chatID, filePath, caption, msgID, username, false)
}

func sendMediaWithProgress(bot *tgbotapi.BotAPI, chatID int64, filePath, caption string, msgID int, username string, isVideo bool) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Failed to stat file: %v", err)
		var msg tgbotapi.Chattable
		if isVideo {
			v := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
			v.SupportsStreaming = true
			if caption != "" {
				v.Caption = caption
				v.ParseMode = "Markdown"
			}
			msg = v
		} else {
			a := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(filePath))
			if caption != "" {
				a.Caption = caption
				a.ParseMode = "Markdown"
			}
			msg = a
		}
		bot.Send(msg)
		return
	}

	fileName := filepath.Base(filePath)
	fileSize := fileInfo.Size()

	var tracker *ui.UploadTracker
	if msgID > 0 && username != "" {
		tracker = ui.NewUploadTracker(bot, chatID, msgID, fileName, fileSize, username)
		log.Printf("Starting upload: %s (%.2f MB)", fileName, float64(fileSize)/(1024*1024))
	}

	progressReader, err := NewProgressReader(filePath, tracker)
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		var msg tgbotapi.Chattable
		if isVideo {
			v := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
			v.SupportsStreaming = true
			if caption != "" {
				v.Caption = caption
				v.ParseMode = "Markdown"
			}
			msg = v
		} else {
			a := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(filePath))
			if caption != "" {
				a.Caption = caption
				a.ParseMode = "Markdown"
			}
			msg = a
		}
		bot.Send(msg)
		return
	}
	defer progressReader.Close()

	var msg tgbotapi.Chattable
	if isVideo {
		video := tgbotapi.NewVideo(chatID, tgbotapi.FileReader{
			Name:   fileName,
			Reader: progressReader,
		})
		video.SupportsStreaming = true
		if caption != "" {
			video.Caption = caption
			video.ParseMode = "Markdown"
		}
		msg = video
	} else {
		audio := tgbotapi.NewAudio(chatID, tgbotapi.FileReader{
			Name:   fileName,
			Reader: progressReader,
		})
		if caption != "" {
			audio.Caption = caption
			audio.ParseMode = "Markdown"
		}
		msg = audio
	}

	startTime := time.Now()
	_, err = bot.Send(msg)

	if err != nil {
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

func sendMediaGroup(bot *tgbotapi.BotAPI, chatID int64, filePaths []string, caption string, isVideo bool) error {
	return sendMediaGroupWithProgress(bot, chatID, filePaths, caption, isVideo, 0, "")
}

func sendMediaGroupWithProgress(bot *tgbotapi.BotAPI, chatID int64, filePaths []string, caption string, isVideo bool, msgID int, username string) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("no files to send")
	}

	if len(filePaths) == 1 {
		if isVideo {
			sendVideoWithProgress(bot, chatID, filePaths[0], caption, msgID, username)
		} else {
			sendAudioWithProgress(bot, chatID, filePaths[0], caption, msgID, username)
		}
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
			if isVideo {
				media := tgbotapi.NewInputMediaVideo(tgbotapi.FilePath(path))
				if i == 0 && j == 0 {
					media.Caption = caption
					media.ParseMode = "Markdown"
				}
				mediaGroup = append(mediaGroup, media)
			} else {
				media := tgbotapi.NewInputMediaAudio(tgbotapi.FilePath(path))
				if i == 0 && j == 0 {
					media.Caption = caption
					media.ParseMode = "Markdown"
				}
				mediaGroup = append(mediaGroup, media)
			}
		}

		mediaGroupConfig := tgbotapi.NewMediaGroup(chatID, mediaGroup)
		if _, err := bot.SendMediaGroup(mediaGroupConfig); err != nil {
			log.Printf("Failed to send media group (batch %d-%d): %v", i, end, err)
			for k, path := range batch {
				if i == 0 && k == 0 {
					if isVideo {
						sendVideoWithProgress(bot, chatID, path, caption, msgID, username)
					} else {
						sendAudioWithProgress(bot, chatID, path, caption, msgID, username)
					}
				} else {
					if isVideo {
						sendVideo(bot, chatID, path)
					} else {
						sendAudio(bot, chatID, path)
					}
				}
			}
		} else {
			log.Printf(" Sent media group: %d files (batch %d-%d)", len(batch), i, end)
		}
	}
	return nil
}
