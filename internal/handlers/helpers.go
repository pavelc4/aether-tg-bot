package handlers

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/config"
)

func isOwner(userID int64) bool {
	ownerID := config.GetOwnerID()
	if ownerID == 0 {
		return false // No owner configured
	}
	return userID == ownerID
}

func DetectSource(url string) string {
	sourceMap := map[string]string{
		"instagram.com":   "Instagram",
		"tiktok.com":      "TikTok",
		"youtube.com":     "YouTube",
		"youtu.be":        "YouTube",
		"twitter.com":     "X (Twitter)",
		"x.com":           "X (Twitter)",
		"facebook.com":    "Facebook",
		"fb.watch":        "Facebook",
		"reddit.com":      "Reddit",
		"pinterest.com":   "Pinterest",
		"soundcloud.com":  "SoundCloud",
		"vimeo.com":       "Vimeo",
		"dailymotion.com": "Dailymotion",
		"twitch.tv":       "Twitch",
		"bilibili.com":    "Bilibili",
		"snapchat.com":    "Snapchat",
		"tumblr.com":      "Tumblr",
		"ok.ru":           "OK.ru",
		"vk.com":          "VK",
	}

	urlLower := strings.ToLower(url)
	for domain, name := range sourceMap {
		if strings.Contains(urlLower, domain) {
			return name
		}
	}
	return "Unknown"
}

func BuildMediaCaption(source, url, mediaType string, size int64, duration time.Duration, username string) string {
	caption := fmt.Sprintf(
		" *Media Downloaded Successfully*\n\n"+
			" *Source:* %s\n"+
			"💾 *Size:* `%s`\n"+
			" *Processing Time:* `%s`\n"+
			" *By:* @%s",
		source,
		FormatFileSize(size),
		formatDuration(duration),
		username,
	)

	return caption
}

func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}


func sendText(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf(" Failed to send text: %v", err)
	}
}

func deleteMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int) {
	del := tgbotapi.NewDeleteMessage(chatID, msgID)
	if _, err := bot.Request(del); err != nil {
		log.Printf("  Failed to delete message: %v", err)
	}
}

func sendVideo(bot *tgbotapi.BotAPI, chatID int64, filePath string) {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
	video.SupportsStreaming = true

	if _, err := bot.Send(video); err != nil {
		log.Printf(" Failed to send video: %v", err)
	}
}

func sendVideoWithCaption(bot *tgbotapi.BotAPI, chatID int64, filePath, caption string) {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
	video.SupportsStreaming = true
	video.Caption = caption
	video.ParseMode = "Markdown"

	if _, err := bot.Send(video); err != nil {
		log.Printf(" Failed to send video with caption: %v", err)
		sendVideo(bot, chatID, filePath)
	}
}

func sendAudio(bot *tgbotapi.BotAPI, chatID int64, filePath string) {
	audio := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(filePath))

	if _, err := bot.Send(audio); err != nil {
		log.Printf(" Failed to send audio: %v", err)
	}
}

func sendAudioWithCaption(bot *tgbotapi.BotAPI, chatID int64, filePath, caption string) {
	audio := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(filePath))
	audio.Caption = caption
	audio.ParseMode = "Markdown"

	if _, err := bot.Send(audio); err != nil {
		log.Printf(" Failed to send audio with caption: %v", err)
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
			log.Printf(" Failed to send media group (batch %d-%d): %v", i, end, err)

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

func formatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
