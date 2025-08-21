package bot

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		HandleHelpCommand(bot, msg)
	case "help":
		HandleHelpCommand(bot, msg)
	case "stats":
		HandleStatusCommand(bot, msg)
	case "support":
		HandleSupportCommand(bot, msg)
	case "tikaudio":
		handleTikTokAudioCommand(bot, msg)
	case "sticker":
		handleStickerCommand(bot, msg)
	default:
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Command tidak dikenal. Ketik /help untuk melihat daftar perintah."))
	}
}

func handleStickerCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Harap sertakan URL stiker.\nContoh: `/sticker https://t.me/addstickers/StickerPackName`"))
		return
	}

	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Memproses stiker, harap tunggu..."))
	defer bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, processingMsg.MessageID))

	re := regexp.MustCompile(`https://t.me/addstickers/(\w+)`)
	matches := re.FindStringSubmatch(args)
	if len(matches) < 2 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ URL stiker tidak valid."))
		return
	}
	packName := matches[1]

	resp, err := bot.Request(tgbotapi.GetStickerSetConfig{Name: packName})
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mendapatkan set stiker."))
		return
	}

	var stickerSet tgbotapi.StickerSet
	if err := json.Unmarshal(resp.Result, &stickerSet); err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal parsing respon stiker."))
		return
	}

	tmpDir, err := os.MkdirTemp("", "stickers-")
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal membuat direktori sementara."))
		return
	}
	defer os.RemoveAll(tmpDir)

	for i, sticker := range stickerSet.Stickers {
		file, err := bot.GetFile(tgbotapi.FileConfig{FileID: sticker.FileID})
		if err != nil {
			log.Printf("Gagal mendapatkan file untuk stiker %s: %v", sticker.FileID, err)
			continue
		}

		ext := filepath.Ext(file.FilePath)
		fileName := fmt.Sprintf("sticker_%d%s", i, ext)
		filePath := filepath.Join(tmpDir, fileName)

		url := file.Link(bot.Token)
		stickerResp, err := http.Get(url)
		if err != nil {
			log.Printf("Gagal mengunduh stiker %s: %v", url, err)
			continue
		}

		out, err := os.Create(filePath)
		if err != nil {
			log.Printf("Gagal membuat file untuk stiker %s: %v", filePath, err)
			stickerResp.Body.Close()
			continue
		}

		_, err = io.Copy(out, stickerResp.Body)
		stickerResp.Body.Close()
		out.Close()
	}

	zipPath := filepath.Join(os.TempDir(), packName+".zip")
	if err := ZipDir(tmpDir, zipPath); err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal membuat file zip."))
		return
	}
	defer os.Remove(zipPath)

	doc := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(zipPath))
	doc.Caption = fmt.Sprintf("Sticker pack: %s", packName)
	bot.Send(doc)
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

	source := "Unknown"
	sourceMap := map[string]string{
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

	for domain, name := range sourceMap {
		if strings.Contains(finalURL, domain) {
			source = name
			break
		}
	}

	bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, fmt.Sprintf("⏳ Sumber terdeteksi: %s. Mengunduh konten...", source)))

	start := time.Now()

	var filePaths []string
	var totalSize int64
	var mediaType string

	resp, err := http.Head(finalURL)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Gagal memeriksa tipe konten: %s", err.Error())))
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	log.Printf("Content-Type for %s: %s", finalURL, contentType)

	switch {
	case strings.HasPrefix(contentType, "image/"):
		mediaType = "Image"
		filePath, size, err := DownloadImage(finalURL)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Gagal mengunduh gambar: %s", err.Error())))
			return
		}
		filePaths = []string{filePath}
		totalSize = size
	case strings.HasPrefix(contentType, "video/"):
		mediaType = "Video"
		paths, size, _, err := DownloadVideo(url)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Gagal mengunduh video: %s", err.Error())))
			return
		}
		filePaths = paths
		totalSize = size
	default: // Default to video download if content type is unknown or not image/video
		mediaType = "Video"
		paths, size, _, err := DownloadVideo(url)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Gagal mengunduh konten: %s", err.Error())))
			return
		}
		filePaths = paths
		totalSize = size
	}

	if len(filePaths) > 1 {
		duration := time.Since(start).Truncate(time.Second)
		caption := BuildMediaCaption(source, finalURL, "Media", totalSize, duration, GetUserName(msg))

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
	} else if len(filePaths) == 1 {
		filePath := filePaths[0]
		err = processAndSendMediaWithMeta(bot, msg, filePath, totalSize, source, start, mediaType, finalURL)
		if err != nil {
			log.Printf("Error processing and sending media: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengirim media: "+err.Error()))
		}
	}

	if len(filePaths) > 0 {
		DeleteDirectory(filepath.Dir(filePaths[0]))
	}
}

func handleDownloadCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Harap sertakan URL setelah perintah.\nContoh: `/mp [URL]`"))
		return
	}
	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Sedang memproses, harap tunggu..."))
	defer bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, processingMsg.MessageID))

	start := time.Now()
	var filePaths []string
	var totalSize int64
	var err error
	var fileType string

	if msg.Command() == "mp" {
		fileType = "Audio"
		filePaths, totalSize, _, err = DownloadAudio(args)
	} else {
		fileType = "Video"
		filePaths, totalSize, _, err = DownloadVideo(args)
	}

	if err != nil {
		errorMsg := fmt.Sprintf("❌ Gagal mengunduh %s: %s", fileType, err.Error())
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		return
	}

	source := "Unknown"
	if strings.Contains(args, "tiktok.com") {
		source = "TikTok"
	} else if strings.Contains(args, "instagram.com") {
		source = "Instagram"
	} else if strings.Contains(args, "facebook.com") {
		source = "Facebook"
	}

	if len(filePaths) > 1 {
		for _, path := range filePaths {
			fileInfo, _ := os.Stat(path)
			processAndSendMediaWithMeta(bot, msg, path, fileInfo.Size(), source, start, fileType, args)
		}
	} else if len(filePaths) == 1 {
		filePath := filePaths[0]
		err = processAndSendMediaWithMeta(bot, msg, filePath, totalSize, source, start, fileType, args)
		if err != nil {
			log.Printf("Error processing and sending media: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengirim media: "+err.Error()))
		}
	}

	if len(filePaths) > 0 {
		DeleteDirectory(filepath.Dir(filePaths[0]))
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

func processAndSendMediaWithMeta(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath string, fileSize int64, source string, start time.Time, fileType string, url string) error {
	ext := filepath.Ext(filePath)
	fileName := filepath.Base(filePath)
	duration := time.Since(start).Truncate(time.Second)

	caption := BuildMediaCaption(source, url, fileType, fileSize, duration, GetUserName(msg))

	switch ext {
	case ".jpg", ".jpeg", ".png":
		imgFile, err := os.Open(filePath)
		if err != nil {
			log.Printf("Error opening image file: %v. Sending as document.", err)
			return sendAsDocument(bot, msg, filePath, caption)
		}
		defer imgFile.Close()

		img, _, err := image.Decode(imgFile)
		if err != nil {
			log.Printf("Error decoding image: %v. Sending as document.", err)
			return sendAsDocument(bot, msg, filePath, caption)
		}

		reencodedFilePath := filepath.Join(filepath.Dir(filePath), "reencoded_"+fileName)
		reencodedFile, err := os.Create(reencodedFilePath)
		if err != nil {
			log.Printf("Error creating re-encoded file: %v. Sending as document.", err)
			return sendAsDocument(bot, msg, filePath, caption)
		}
		defer reencodedFile.Close()
		defer os.Remove(reencodedFilePath)

		if ext == ".jpg" || ext == ".jpeg" {
			err = jpeg.Encode(reencodedFile, img, &jpeg.Options{Quality: 90})
		} else {
			err = png.Encode(reencodedFile, img)
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

	case ".mp4", ".webm", ".mov":
		video := tgbotapi.NewVideo(msg.Chat.ID, tgbotapi.FilePath(filePath))
		video.Caption = caption
		video.ParseMode = "MarkdownV2"
		if _, err := bot.Send(video); err != nil {
			log.Printf("Error sending video: %v. Falling back to document.", err)
			return sendAsDocument(bot, msg, filePath, caption)
		}
		return nil

	default:
		return sendAsDocument(bot, msg, filePath, caption)
	}
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
func handleTikTokAudioCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	url := strings.TrimSpace(msg.CommandArguments())
	if url == "" || !strings.Contains(url, "tiktok.com") {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Harap sertakan URL TikTok setelah perintah.\nContoh: `/tikaudio {URL}`"))
		return
	}

	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Memproses audio TikTok, harap tunggu..."))
	defer bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, processingMsg.MessageID))

	start := time.Now()

	filePath, title, author, err := DownloadTikTokAudio(url)
	if err != nil {
		errorMsg := fmt.Sprintf("❌ Gagal mengunduh audio: %s", err.Error())
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		return
	}

	defer DeleteDirectory(filepath.Dir(filePath))

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Gagal mendapatkan info file: %v", err)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Maaf, gagal mendapatkan informasi file."))
		return
	}
	fileSize := fileInfo.Size()
	duration := time.Since(start).Truncate(time.Second)

	audioFile := tgbotapi.FilePath(filePath)
	audioMsg := tgbotapi.NewAudio(msg.Chat.ID, audioFile)

	audioMsg.Caption = BuildMediaCaption("TikTok", url, "Audio", fileSize, duration, GetUserName(msg))
	audioMsg.ParseMode = "MarkdownV2"
	audioMsg.Title = title
	audioMsg.Performer = author

	if _, err := bot.Send(audioMsg); err != nil {
		log.Printf("Gagal mengirim audio ke Telegram: %v", err)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Maaf, gagal mengirim file audio."))
	}
}
