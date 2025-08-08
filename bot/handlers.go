package bot

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCommand memproses perintah yang dikirim ke bot
func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Harap sertakan URL setelah command"))
		return
	}

	start := time.Now()

	switch msg.Command() {
	case "l":
		filePath, fileSize, provider, err := DownloadVideo(args)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal download video: "+err.Error()))
			return
		}
		sendFileWithMeta(bot, msg, filePath, fileSize, provider, start, "Video")

	case "mp":
		filePath, fileSize, provider, err := DownloadAudio(args)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal download audio: "+err.Error()))
			return
		}
		sendFileWithMeta(bot, msg, filePath, fileSize, provider, start, "Audio")

	default:
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Command tidak dikenal"))
	}
}

// sendFileWithMeta mengirim file beserta metadata seperti ukuran, durasi, dan provider
func sendFileWithMeta(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath string, fileSize int64, provider string, start time.Time, fileType string) {
	duration := time.Since(start)

	caption := fmt.Sprintf(
		"🔗 Sumber: [Klik Disini](%s)\n🏷 Tipe: %s\n💾 Ukuran: %s\n⏱️ Durasi Proses: %s\n👤 Oleh: @%s\n📡 Provider: %s",
		msg.CommandArguments(),
		fileType,
		FormatFileSize(fileSize),
		duration.Truncate(time.Second),
		msg.From.UserName,
		provider,
	)

	doc := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filePath))
	doc.Caption = caption
	doc.ParseMode = "Markdown"

	if _, err := bot.Send(doc); err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengirim file: "+err.Error()))
	}
}
