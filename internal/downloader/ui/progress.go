package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func BuildProgressBar(percentage float64) string {
	const barLength = 10
	const filledChar = "■"
	const emptyChar = "□"

	filled := int(percentage / 100 * float64(barLength))
	if filled > barLength {
		filled = barLength
	}
	if filled < 0 {
		filled = 0
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

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := bot.Send(edit)
		if err == nil {
			log.Printf(" [Progress] Updated: %.1f%% - %s at %s", progress.Percentage, progress.Downloaded, progress.Speed)
			return
		}

		if strings.Contains(err.Error(), "Too Many Requests") || strings.Contains(err.Error(), "retry after") {
			waitTime := time.Duration(attempt+1) * 2 * time.Second
			log.Printf(" [Progress] Rate limited, waiting %v before retry %d/%d", waitTime, attempt+1, maxRetries)
			time.Sleep(waitTime)
			continue
		}

		if attempt == maxRetries-1 {
			log.Printf(" [Progress] Failed to update after %d attempts: %v", maxRetries, err)
		}
	}
}

func UpdateInitialProgressMessageDetailed(bot *tgbotapi.BotAPI, chatID int64, msgID int, fileName string, totalSize string, platform string, username string) {
	if bot == nil {
		log.Printf(" [Progress] bot is nil, skipping initial message")
		return
	}

	text := fmt.Sprintf(
		"📄 %s\n"+
			"├ [ □□□□□□□□□□ 0.0%%]\n"+
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
		log.Printf("bot is nil, skipping update")
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
		log.Printf("Failed to update: %v", err)
		return
	}
}

func UpdateInitialProgressMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int, platform string) {
	if bot == nil {
		log.Printf("bot is nil, skipping initial message")
		return
	}

	text := fmt.Sprintf(
		"📥 *Downloading from %s*\n\n"+
			"[□□□□□□□□□□ 0.0%%]\n"+
			"├─ 📦 Size: `Starting...`\n"+
			"├─ 🚀 Speed: `--`\n"+
			"└─ ⏱️  Elapsed: `--`",
		platform,
	)

	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.ParseMode = "Markdown"

	if _, err := bot.Send(edit); err != nil {
		log.Printf("Failed to update initial: %v", err)
		return
	}

	log.Printf("Sent initial message")
}

func UpdateUploadProgressMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int, fileName string, uploadProgress UploadProgress, username string) {
	if bot == nil {
		log.Printf("⚠️ [Upload] bot is nil, skipping update")
		return
	}

	progressBar := BuildProgressBar(uploadProgress.Percentage)
	text := fmt.Sprintf(
		"📄 %s\n"+
			"├ %s\n"+
			"├ Status: 📤 Uploading to Telegram...\n"+
			"├ Uploaded: %s / %s\n"+
			"├ Speed: %s\n"+
			"└ User: @%s",
		fileName,
		progressBar,
		uploadProgress.Uploaded,
		uploadProgress.TotalSize,
		uploadProgress.Speed,
		username,
	)

	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	if _, err := bot.Send(edit); err != nil {
		if !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("⚠️ [Upload] Update failed: %v", err)
		}
	}
}

func UpdateUploadCompleteMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int, fileName string, totalSize string, duration string, username string) {
	if bot == nil {
		return
	}

	text := fmt.Sprintf(
		"✅ Upload Complete!\n\n"+
			"📄 %s\n"+
			"├ Size: %s\n"+
			"├ Duration: %s\n"+
			"└ User: @%s",
		fileName,
		totalSize,
		duration,
		username,
	)

	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	if _, err := bot.Send(edit); err != nil {
		log.Printf("⚠️ [Upload] Failed to update complete message: %v", err)
	}
}
