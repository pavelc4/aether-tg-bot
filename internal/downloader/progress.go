package downloader

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// DownloadProgress represents download progress information
type DownloadProgress struct {
	Percentage float64
	Downloaded string
	Speed      string
	ETA        string
	Status     string
}

// UpdateProgressMessage updates progress message in Telegram
func UpdateProgressMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int, platform string, progress DownloadProgress) {
	if bot == nil {
		return
	}

	text := fmt.Sprintf(
		"⏳ *Downloading from %s*\n"+
			"━━━━━━━━━━━━━━━━━\n"+
			"📊 Progress: `%.1f%%`\n"+
			"📦 Downloaded: `%s`\n"+
			"⚡ Speed: `%s`\n"+
			"⏱ ETA: `%s`\n"+
			"━━━━━━━━━━━━━━━━━",
		platform,
		progress.Percentage,
		progress.Downloaded,
		progress.Speed,
		progress.ETA,
	)

	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.ParseMode = "Markdown"

	if _, err := bot.Send(edit); err != nil {
		log.Printf("⚠️  Failed to update progress: %v", err)
	}
}
