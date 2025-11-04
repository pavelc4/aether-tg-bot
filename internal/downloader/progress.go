package downloader

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type DownloadProgress struct {
	Percentage float64
	Downloaded string
	Speed      string
	ETA        string
	Status     string
}

func BuildProgressBar(percentage float64) string {
	const barLength = 10
	const filledChar = "■"
	const emptyChar = "□"

	filled := int(percentage / 100 * float64(barLength))
	if filled > barLength {
		filled = barLength
	}

	bar := strings.Repeat(filledChar, filled) + strings.Repeat(emptyChar, barLength-filled)
	return fmt.Sprintf("[%s] %.1f%%", bar, percentage)
}

func UpdateProgressMessageDetailed(bot *tgbotapi.BotAPI, chatID int64, msgID int, fileName string, progress DownloadProgress, totalSize string, platform string, username string) {
	if bot == nil {
		log.Printf(" [Progress] bot is nil, skipping update")
		return
	}

	progressBar := BuildProgressBar(progress.Percentage)

	text := fmt.Sprintf(
		"📄 %s\n"+
			"├ %s\n"+
			"├ Status: 📥 Downloading...\n"+
			"├ Ukuran: %s / %s\n"+
			"├ Kecepatan: %s\n"+
			"├ Estimasi: %s\n"+
			"├ Engine: yt-dlp\n"+
			"└ Oleh: @%s",
		fileName,
		progressBar,
		progress.Downloaded,
		totalSize,
		progress.Speed,
		progress.ETA,
		username,
	)

	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)

	if _, err := bot.Send(edit); err != nil {
		log.Printf(" [Progress] Failed to update detailed: %v", err)
		return
	}

	log.Printf(" [Progress] Updated: %.1f%% - %s at %s", progress.Percentage, progress.Downloaded, progress.Speed)
}

func UpdateInitialProgressMessageDetailed(bot *tgbotapi.BotAPI, chatID int64, msgID int, fileName string, totalSize string, platform string, username string) {
	if bot == nil {
		log.Printf(" [Progress] bot is nil, skipping initial message")
		return
	}

	text := fmt.Sprintf(
		"📄 %s\n"+
			"├ [□□□□□□□□□□□ 0.0%%]\n"+
			"├ Status: 📥 Downloading...\n"+
			"├ Size: 0 / %s\n"+
			"├ Speed: --\n"+
			"├ estimate : --\n"+
			"├ Engine: yt-dlp\n"+
			"└ User: @%s",
		fileName,
		totalSize,
		username,
	)

	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)

	if _, err := bot.Send(edit); err != nil {
		log.Printf(" [Progress] Failed to update initial detailed: %v", err)
		return
	}

	log.Printf(" [Progress] Sent initial detailed message")
}

func UpdateProgressMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int, platform string, progress DownloadProgress) {
	if bot == nil {
		log.Printf(" [Progress] bot is nil, skipping update")
		return
	}

	progressBar := BuildProgressBar(progress.Percentage)

	text := fmt.Sprintf(
		"📥 *Downloading from %s*\n\n"+
			"%s\n"+
			"├─ 📦 Size: `%s`\n"+
			"├─ 🚀 Speed: `%s`\n"+
			"└─ ⏱️  Elapsed: `%s`",
		platform,
		progressBar,
		progress.Downloaded,
		progress.Speed,
		progress.ETA,
	)

	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.ParseMode = "Markdown"

	if _, err := bot.Send(edit); err != nil {
		log.Printf(" [Progress] Failed to update: %v", err)
		return
	}

	log.Printf(" [Progress] Updated: %.1f%% - %s at %s", progress.Percentage, progress.Downloaded, progress.Speed)
}

func UpdateInitialProgressMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int, platform string) {
	if bot == nil {
		log.Printf(" [Progress] bot is nil, skipping initial message")
		return
	}

	text := fmt.Sprintf(
		"📥 *Downloading from %s*\n\n"+
			"[□□□□□□□□□□□ 0.0%%]\n"+
			"├─ 📦 Size: `Starting...`\n"+
			"├─ 🚀 Speed: `--`\n"+
			"└─ ⏱️  Elapsed: `--`",
		platform,
	)

	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.ParseMode = "Markdown"

	if _, err := bot.Send(edit); err != nil {
		log.Printf(" [Progress] Failed to update initial: %v", err)
		return
	}

	log.Printf(" [Progress] Sent initial message")
}
