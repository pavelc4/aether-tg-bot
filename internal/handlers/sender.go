package handlers

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func sendText(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("❌ Failed to send text: %v", err)
	}
}

func deleteMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int) {
	del := tgbotapi.NewDeleteMessage(chatID, msgID)
	if _, err := bot.Request(del); err != nil {
		log.Printf("❌ Failed to delete message: %v", err)
	}
}

func sendVideo(bot *tgbotapi.BotAPI, chatID int64, filePath string) {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
	video.SupportsStreaming = true
	if _, err := bot.Send(video); err != nil {
		log.Printf("❌ Failed to send video: %v", err)
	}
}

func sendVideoWithCaption(bot *tgbotapi.BotAPI, chatID int64, filePath, caption string) {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
	video.SupportsStreaming = true
	video.Caption = caption
	video.ParseMode = "Markdown"
	if _, err := bot.Send(video); err != nil {
		log.Printf("❌ Failed to send video with caption: %v", err)
		sendVideo(bot, chatID, filePath)
	}
}

func sendAudio(bot *tgbotapi.BotAPI, chatID int64, filePath string) {
	audio := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(filePath))
	if _, err := bot.Send(audio); err != nil {
		log.Printf("❌ Failed to send audio: %v", err)
	}
}

func sendAudioWithCaption(bot *tgbotapi.BotAPI, chatID int64, filePath, caption string) {
	audio := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(filePath))
	audio.Caption = caption
	audio.ParseMode = "Markdown"
	if _, err := bot.Send(audio); err != nil {
		log.Printf("❌ Failed to send audio with caption: %v", err)
		sendAudio(bot, chatID, filePath)
	}
}

func sendMediaGroup(bot *tgbotapi.BotAPI, chatID int64, filePaths []string, caption string, isVideo bool) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("no files to send")
	}

	if len(filePaths) == 1 {
		if isVideo {
			sendVideoWithCaption(bot, chatID, filePaths[0], caption)
		} else {
			sendAudioWithCaption(bot, chatID, filePaths[0], caption)
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
			log.Printf("❌ Failed to send media group (batch %d-%d): %v", i, end, err)
			for k, path := range batch {
				if i == 0 && k == 0 {
					if isVideo {
						sendVideoWithCaption(bot, chatID, path, caption)
					} else {
						sendAudioWithCaption(bot, chatID, path, caption)
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
