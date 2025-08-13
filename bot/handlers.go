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
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

func formatFileSize(size int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d Bytes", size)
	}
}

func formatUptime(uptimeSec uint64) string {
	days := uptimeSec / (60 * 60 * 24)
	hours := (uptimeSec % (60 * 60 * 24)) / (60 * 60)
	minutes := (uptimeSec % (60 * 60)) / 60
	seconds := uptimeSec % 60

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

var markdownV2Replacer = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
	"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
	"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
)

func escapeMarkdownV2(s string) string {
	return markdownV2Replacer.Replace(s)
}

func buildMediaCaption(source, url, fileType string, fileSize int64, duration time.Duration, user string) string {
	escapedSource := escapeMarkdownV2(strings.ToLower(source))
	escapedURL := escapeMarkdownV2(url)
	escapedFileType := escapeMarkdownV2(fileType)
	escapedSize := escapeMarkdownV2(formatFileSize(fileSize))
	escapedDuration := escapeMarkdownV2(duration.String())
	escapedUser := escapeMarkdownV2(user)

	captionFormat := `✅ *%s Berhasil Diunduh\!*` + "\n\n" +
		`🔗 *Sumber:* [%s](%s)` + "\n" +
		`💾 *Ukuran:* %s` + "\n" +
		`⏱️ *Durasi Proses:* %s` + "\n" +
		`👤 *Oleh:* %s`

	return fmt.Sprintf(
		captionFormat,
		escapedFileType,
		escapedSource,
		escapedURL,
		escapedSize,
		escapedDuration,
		escapedUser,
	)
}

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start", "help":
		handleHelpCommand(bot, msg)
	case "stats":
		handleStatusCommand(bot, msg)
	default:
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Command tidak dikenal. Ketik /help untuk melihat daftar perintah."))
	}
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	re := regexp.MustCompile(`(https?://[^\s]+)`)
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
	if strings.Contains(finalURL, "tiktok.com") {
		source = "TikTok"
	} else if strings.Contains(finalURL, "instagram.com") {
		source = "Instagram"
	} else if strings.Contains(finalURL, "facebook.com") {
		source = "Facebook"
	}

	bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, fmt.Sprintf("⏳ Sumber terdeteksi: %s. Mengunduh konten...", source)))

	start := time.Now()

	filePaths, totalSize, _, err := DownloadVideo(url)
	if err != nil {
		errorMsg := fmt.Sprintf("❌ Gagal mengunduh video: %s", err.Error())
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		return
	}

	if len(filePaths) > 1 {
		duration := time.Since(start).Truncate(time.Second)
		caption := buildMediaCaption(source, finalURL, "Media", totalSize, duration, GetUserName(msg))

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
		err = processAndSendMediaWithMeta(bot, msg, filePath, totalSize, source, start, "Video", finalURL)
		if err != nil {
			log.Printf("Error processing and sending media: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengirim media: "+err.Error()))
		}
	}

	if len(filePaths) > 0 {
		DeleteDirectory(filepath.Dir(filePaths[0]))
	}
}

func handleHelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	helpText := "Selamat datang di Aether Bot! ✨\n\n" +
		"Cukup kirimkan link dari TikTok atau Instagram, dan saya akan mengunduh kontennya untuk Anda.\n\n" +
		"Perintah yang tersedia:\n" +
		" • `/help` - Menampilkan pesan ini.\n" +
		" • `/stats` - Menampilkan status bot."
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, helpText))
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

	caption := buildMediaCaption(source, url, "Media", totalSize, duration, GetUserName(msg))

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, caption)
	msgConfig.ParseMode = "MarkdownV2"
	bot.Send(msgConfig)
}

func processAndSendMediaWithMeta(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath string, fileSize int64, source string, start time.Time, fileType string, url string) error {
	ext := filepath.Ext(filePath)
	fileName := filepath.Base(filePath)
	duration := time.Since(start).Truncate(time.Second)

	caption := buildMediaCaption(source, url, fileType, fileSize, duration, GetUserName(msg))

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

func handleStatusCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	hostInfo, _ := host.Info()
	cpuCounts, _ := cpu.Counts(true)
	cpuUsage, _ := cpu.Percent(time.Second, false)
	ramInfo, _ := mem.VirtualMemory()
	diskInfo, _ := disk.Usage("/")
	netIO, _ := net.IOCounters(false)

	var totalTraffic, bytesSent, bytesRecv uint64
	if len(netIO) > 0 {
		bytesSent = netIO[0].BytesSent
		bytesRecv = netIO[0].BytesRecv
		totalTraffic = bytesSent + bytesRecv
	}

	proc, _ := process.NewProcess(int32(os.Getpid()))
	procRAMInfo, _ := proc.MemoryInfo()

	statusText := fmt.Sprintf(
		"⚙️ *System:*\n"+
			"├─ CPU: `%.2f%%` `(%d-core)`\n"+
			"├─ RAM: `%s / %s` `(%.2f%%)`\n"+
			"├─ Disk: `%s / %s` `(%.2f%%)`\n"+
			"└─ Uptime: `%s`\n\n"+
			"🐹 *App:*\n"+
			"└─ RAM Usage: `%s`\n\n"+
			"🌐 *Networks:*\n"+
			"├─ In: `%s`\n"+
			"├─ Out: `%s`\n"+
			"└─ Total Traffic: `%s`",
		cpuUsage[0], cpuCounts,
		formatFileSize(int64(ramInfo.Used)), formatFileSize(int64(ramInfo.Total)), ramInfo.UsedPercent,
		formatFileSize(int64(diskInfo.Used)), formatFileSize(int64(diskInfo.Total)), diskInfo.UsedPercent,
		formatUptime(hostInfo.Uptime),
		formatFileSize(int64(procRAMInfo.RSS)),
		formatFileSize(int64(bytesRecv)),
		formatFileSize(int64(bytesSent)),
		formatFileSize(int64(totalTraffic)),
	)

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, statusText)
	msgConfig.ParseMode = "MarkdownV2"
	bot.Send(msgConfig)
}
