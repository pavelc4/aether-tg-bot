package bot

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start", "help":
		handleHelpCommand(bot, msg)
	case "l", "mp":
		handleDownloadCommand(bot, msg)
	case "img":
		handleImageCommand(bot, msg)
	default:
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Command tidak dikenal. Ketik /help untuk melihat daftar perintah."))
	}
}

func handleHelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	helpText := "Selamat datang di Aether Bot! ✨\n\n" +
		"Gunakan perintah berikut:\n" +
		" • `/mp [URL]` - Untuk mengunduh audio (MP3).\n" +
		" • `/l [URL]` - Untuk mengunduh video.\n" +
		" • `/img [URL]` - Untuk mengunduh gambar dari sosial media."
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, helpText))
}

func handleImageCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	url := strings.TrimSpace(msg.CommandArguments())
	if url == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Harap sertakan URL postingan.\nContoh: `/img [URL]`"))
		return
	}

	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Memproses link, harap tunggu..."))

	finalURL, err := ResolveFinalURL(url)
	if err != nil {
		bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, processingMsg.MessageID))
		errorMsg := fmt.Sprintf("❌ Gagal memproses link: %s", err.Error())
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		return
	}

	if strings.Contains(finalURL, "facebook.com/groups/") {
		bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, "⏳ Link grup terdeteksi, mencoba metode khusus..."))

		filePath, err := ScrapeFacebookGroup(finalURL)
		bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, processingMsg.MessageID))

		if err != nil {
			errorMsg := fmt.Sprintf("❌ Gagal mengambil gambar dari grup: %s", err.Error())
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
			return
		}

		parentDir := filepath.Dir(filePath)
		defer DeleteDirectory(parentDir)
		photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FilePath(filePath))
		bot.Send(photo)

	} else {
		bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, "⏳ Link ditemukan, sedang mengambil gambar..."))

		filePaths, err := DownloadImagesFromURL(finalURL)
		bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, processingMsg.MessageID))

		if err != nil {
			errorMsg := fmt.Sprintf("❌ Gagal mengambil gambar: %s", err.Error())
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
			return
		}

		if len(filePaths) > 0 {
			parentDir := filepath.Dir(filePaths[0])
			defer DeleteDirectory(parentDir)
		}

		if len(filePaths) > 1 {
			mediaGroup := []interface{}{}
			for i, path := range filePaths {
				if i >= 10 {
					break
				}
				mediaGroup = append(mediaGroup, tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(path)))
			}
			group := tgbotapi.NewMediaGroup(msg.Chat.ID, mediaGroup)
			bot.Send(group)
		} else if len(filePaths) == 1 {
			photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FilePath(filePaths[0]))
			bot.Send(photo)
		}
	}
}

func handleDownloadCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Harap sertakan URL setelah perintah.\nContoh: `/mp [URL]`"))
		return
	}
	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Sedang memproses, harap tunggu..."))

	start := time.Now()
	var filePath string
	var fileSize int64
	var provider string
	var err error
	var fileType string

	if msg.Command() == "mp" {
		fileType = "Audio"
		filePath, fileSize, provider, err = DownloadAudio(args)
	} else {
		fileType = "Video"
		filePath, fileSize, provider, err = DownloadVideo(args)
	}

	bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, processingMsg.MessageID))

	if err != nil {
		errorMsg := fmt.Sprintf("❌ Gagal mengunduh %s: %s", fileType, err.Error())
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		return
	}

	parentDir := filepath.Dir(filePath)
	defer DeleteDirectory(parentDir)

	sendFileWithMeta(bot, msg, filePath, fileSize, provider, start, fileType)
}

func sendFileWithMeta(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath string, fileSize int64, provider string, start time.Time, fileType string) {
	duration := time.Since(start).Truncate(time.Second)

	caption := fmt.Sprintf(
		"✅ *%s Berhasil Diunduh!*\n\n"+
			"🔗 *Sumber:* [Klik Disini](%s)\n"+
			"💾 *Ukuran:* %s\n"+
			"⏱️ *Durasi Proses:* %s\n"+
			"👤 *Oleh:* %s",
		fileType,
		msg.CommandArguments(),
		FormatFileSize(fileSize),
		duration,
		GetUserName(msg),
	)

	doc := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filePath))
	doc.Caption = caption
	doc.ParseMode = "Markdown"

	if _, err := bot.Send(doc); err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengirim file: "+err.Error()))
	}
}
