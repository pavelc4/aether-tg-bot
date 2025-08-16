package bot

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CommandHandler func(bot *tgbotapi.BotAPI, msg *tgbotapi.Message)

var commandHandlers = map[string]CommandHandler{
	"start":   handleHelpCommand,
	"help":    handleHelpCommand,
	"stats":   HandleStatusCommand,
	"support": HandleSupportCommand,
	"mp3":     handleDownloadCommand,
	"mp4":     handleDownloadCommand,
}

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if handler, ok := commandHandlers[msg.Command()]; ok {
		handler(bot, msg)
	} else {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Command tidak dikenal. Ketik /help untuk melihat daftar perintah."))
	}
}

var sourceMap = map[string]string{
	"bilibili.com":    "Bilibili",
	"bluesky.app":     "Bluesky",
	"dailymotion.com": "Dailymotion",
	"facebook.com":    "Facebook",
	"instagram.com":   "Instagram",
	"loom.com":        "Loom",
	"ok.ru":           "OK",
	"pinterest.com":   "Pinterest",
	"newgrounds.com":  "Newgrounds",
	"reddit.com":      "Reddit",
	"rutube.ru":       "Rutube",
	"snapchat.com":    "Snapchat",
	"soundcloud.com":  "Soundcloud",
	"streamable.com":  "Streamable",
	"tiktok.com":      "TikTok",
	"tumblr.com":      "Tumblr",
	"twitch.tv":       "Twitch",
	"twitter.com":     "Twitter",
	"vimeo.com":       "Vimeo",
	"vk.com":          "VK",
	"xiaohongshu.com": "Xiaohongshu",
	"youtube.com":     "YouTube",
}

func detectSource(url string) string {
	for domain, name := range sourceMap {
		if strings.Contains(url, domain) {
			return name
		}
	}
	return "Unknown"
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	re := regexp.MustCompile(`(https?://[^
]+)`)
	url := re.FindString(msg.Text)

	if url == "" {
		return
	}
	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Memproses link, harap tunggu..."))
	defer bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, processingMsg.MessageID))

	finalURL, err := ResolveFinalURL(url)
	if err != nil {
		errorMsg := fmt.Sprintf("❌ Gagal memproses link: %s", err.Error())
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		return
	}

	source := detectSource(finalURL)
	bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, fmt.Sprintf("⏳ Sumber terdeteksi: %s. Mengunduh konten...", source)))

	start := time.Now()

	filePaths, totalSize, _, err := DownloadVideo(url)
	if err != nil {
		errorMsg := fmt.Sprintf("❌ Gagal mengunduh video: %s", err.Error())
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		return
	}

	sendMedia(bot, msg, filePaths, totalSize, source, start, "Video", finalURL)
}

func handleHelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	helpText := "Selamat datang di Aether Bot! ✨\n\n" +
		"Saya adalah bot yang dapat membantu Anda mengunduh media dari berbagai platform sosial media.\n\n" +
		"Cukup kirimkan link dari platform yang didukung, dan saya akan mengunduh kontennya untuk Anda.\n\n" +
		"Platform yang didukung:\n" +
		"- Bilibili\n" +
		"- Bluesky\n" +
		"- Dailymotion\n" +
		"- Facebook\n" +
		"- Instagram\n" +
		"- Loom\n" +
		"- OK\n" +
		"- Pinterest\n" +
		"- Newgrounds\n" +
		"- Reddit\n" +
		"- Rutube\n" +
		"- Snapchat\n" +
		"- Soundcloud\n" +
		"- Streamable\n" +
		"- TikTok\n" +
		"- Tumblr\n" +
		"- Twitch\n" +
		"- Twitter\n" +
		"- Vimeo\n" +
		"- VK\n" +
		"- Xiaohongshu\n" +
		"- YouTube\n\n" +
		"Perintah yang tersedia:\n" +
		" • `/help` - Menampilkan pesan ini.\n" +
		" • `/stats` - Menampilkan status bot."

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Developer", "https://t.me/pavellc"),
		),
	)

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, helpText)
	msgConfig.ReplyMarkup = inlineKeyboard
	bot.Send(msgConfig)
}

func handleDownloadCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Harap sertakan URL setelah perintah.\nContoh: `/mp3 [URL]`"))
		return
	}
	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Sedang memproses, harap tunggu..."))
	defer bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, processingMsg.MessageID))

	start := time.Now()
	var filePaths []string
	var totalSize int64
	var err error
	fileType := "Video"

	if msg.Command() == "mp3" {
		fileType = "Audio"
		filePaths, totalSize, _, err = DownloadAudio(args)
	} else {
		filePaths, totalSize, _, err = DownloadVideo(args)
	}

	if err != nil {
		errorMsg := fmt.Sprintf("❌ Gagal mengunduh %s: %s", fileType, err.Error())
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		return
	}

	source := detectSource(args)
	sendMedia(bot, msg, filePaths, totalSize, source, start, fileType, args)
}

func sendMedia(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePaths []string, totalSize int64, source string, start time.Time, fileType string, url string) {
	if len(filePaths) == 0 {
		return
	}
	defer DeleteDirectory(filepath.Dir(filePaths[0]))

	if len(filePaths) > 1 {
		duration := time.Since(start).Truncate(time.Second)
		caption := BuildMediaCaption(source, url, "Media", totalSize, duration, GetUserName(msg))

		mediaGroup := []interface{}{}
		for i, path := range filePaths {
			if i >= 10 {
				break
			}

			photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(path))
			if i == 0 {
				photo.Caption = caption
				photo.ParseMode = "MarkdownV2"
			}
			mediaGroup = append(mediaGroup, photo)
		}
		if len(mediaGroup) > 0 {
			group := tgbotapi.NewMediaGroup(msg.Chat.ID, mediaGroup)
			if _, err := bot.Request(group); err != nil {
				log.Printf("Error sending media group to Telegram: %v", err)
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengirim media group: "+err.Error()))
			}
		}
	} else {
		filePath := filePaths[0]
		err := processAndSendMediaWithMeta(bot, msg, filePath, totalSize, source, start, fileType, url)
		if err != nil {
			log.Printf("Error processing and sending media: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengirim media: "+err.Error()))
		}
	}
}

func sendDetailedMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, source string, start time.Time, filePaths []string, url string) {
	duration := time.Since(start).Truncate(time.Second)

	var totalSize int64
	for _, path := range filePaths {
		if fileInfo, err := os.Stat(path); err == nil {
			totalSize += fileInfo.Size()
		}
	}

	caption := BuildMediaCaption(source, url, "Media", totalSize, duration, GetUserName(msg))

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, caption)
	msgConfig.ParseMode = "MarkdownV2"
	bot.Send(msgConfig)
}

type MediaSender func(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath, caption string) error

var mediaSenders = map[string]MediaSender{
	".jpg":  sendAsPhoto,
	".jpeg": sendAsPhoto,
	".png":  sendAsPhoto,
	".mp4":  sendAsVideo,
	".webm": sendAsVideo,
	".mov":  sendAsVideo,
}

func processAndSendMediaWithMeta(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath string, fileSize int64, source string, start time.Time, fileType string, url string) error {
	ext := filepath.Ext(filePath)
	duration := time.Since(start).Truncate(time.Second)
	caption := BuildMediaCaption(source, url, fileType, fileSize, duration, GetUserName(msg))

	if sender, ok := mediaSenders[ext]; ok {
		return sender(bot, msg, filePath, caption)
	}
	return sendAsDocument(bot, msg, filePath, caption)
}

func sendAsPhoto(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath, caption string) error {
	imgFile, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening image file: %v. Sending as document.", err)
		return sendAsDocument(bot, msg, filePath, caption)
	}
	defer imgFile.Close()

	img, format, err := image.Decode(imgFile)
	if err != nil {
		log.Printf("Error decoding image: %v. Sending as document.", err)
		return sendAsDocument(bot, msg, filePath, caption)
	}

	reencodedFilePath := filepath.Join(filepath.Dir(filePath), "reencoded_"+filepath.Base(filePath))
	reencodedFile, err := os.Create(reencodedFilePath)
	if err != nil {
		log.Printf("Error creating re-encoded file: %v. Sending as document.", err)
		return sendAsDocument(bot, msg, filePath, caption)
	}
	defer reencodedFile.Close()
	defer os.Remove(reencodedFilePath)

	switch format {
	case "jpeg":
		err = jpeg.Encode(reencodedFile, img, &jpeg.Options{Quality: 90})
	case "png":
		err = png.Encode(reencodedFile, img)
	default:
		log.Printf("Unsupported image format: %s. Sending as document.", format)
		return sendAsDocument(bot, msg, filePath, caption)
	}

	if err != nil {
		log.Printf("Error re-encoding image: %v. Sending as document.", err)
		return sendAsDocument(bot, msg, filePath, caption)
	}

	photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FilePath(reencodedFilePath))
	photo.Caption = caption
	photo.ParseMode = "MarkdownV2"
	if _, err := bot.Send(photo); err != nil {
		log.Printf("Error sending re-encoded photo: %v. Falling back to document.", err)
		return sendAsDocument(bot, msg, reencodedFilePath, caption)
	}
	return nil
}

func sendAsVideo(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath, caption string) error {
	video := tgbotapi.NewVideo(msg.Chat.ID, tgbotapi.FilePath(filePath))
	video.Caption = caption
	video.ParseMode = "MarkdownV2"
	if _, err := bot.Send(video); err != nil {
		log.Printf("Error sending video: %v. Falling back to document.", err)
		return sendAsDocument(bot, msg, filePath, caption)
	}
	return nil
}

func sendAsDocument(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath, caption string) error {
	doc := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filePath))
	doc.Caption = caption
	doc.ParseMode = "MarkdownV2"
	if _, err := bot.Send(doc); err != nil {
		return fmt.Errorf("failed to send file as document: %w", err)
	}
	return nil
}
