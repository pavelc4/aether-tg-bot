package bot

import (
	"fmt"
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

	if strings.Contains(finalURL, "instagram.com/p/") || strings.Contains(finalURL, "instagram.com/reel/") {
		filePaths, err := DownloadImagesFromURL(finalURL)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengambil konten dari Instagram: "+err.Error()))
			return
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
		if len(filePaths) > 0 {
			DeleteDirectory(filepath.Dir(filePaths[0]))
		}
	} else {
		filePath, fileSize, _, err := DownloadVideo(url)
		if err != nil {
			errorMsg := fmt.Sprintf("❌ Gagal mengunduh video: %s", err.Error())
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
			return
		}
		sendFileWithMeta(bot, msg, filePath, fileSize, source, start, "Video", url)
		DeleteDirectory(filepath.Dir(filePath))
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
	var filePath string
	var fileSize int64
	var err error
	var fileType string

	if msg.Command() == "mp" {
		fileType = "Audio"
		filePath, fileSize, _, err = DownloadAudio(args)
	} else {
		fileType = "Video"
		filePath, fileSize, _, err = DownloadVideo(args)
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

	sendFileWithMeta(bot, msg, filePath, fileSize, source, start, fileType, args)
	DeleteDirectory(filepath.Dir(filePath))
}
func sendFileWithMeta(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath string, fileSize int64, source string, start time.Time, fileType string, url string) {
	duration := time.Since(start).Truncate(time.Second)

	escapedSource := escapeMarkdownV2(strings.ToLower(source))
	escapedURL := escapeMarkdownV2(url)
	escapedFileType := escapeMarkdownV2(fileType)
	escapedSize := escapeMarkdownV2(formatFileSize(fileSize))
	escapedDuration := escapeMarkdownV2(duration.String())
	escapedUser := escapeMarkdownV2(GetUserName(msg))

	caption := fmt.Sprintf(
		`✅ *%s Berhasil Diunduh\!*

		🔗 *Sumber:* [%s](%s)
		💾 *Ukuran:* %s
		⏱️ *Durasi Proses:* %s
		👤 *Oleh:* %s`,
		escapedFileType,
		escapedSource,
		escapedURL,
		escapedSize,
		escapedDuration,
		escapedUser,
	)

	doc := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filePath))
	doc.Caption = caption
	doc.ParseMode = "MarkdownV2"
	if _, err := bot.Send(doc); err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengirim file: "+err.Error()))
	}
}
func handleImageCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	url := strings.TrimSpace(msg.CommandArguments())
	if url == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Harap sertakan URL postingan.\nContoh: `/img [URL]`"))
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

	if strings.Contains(finalURL, "facebook.com/groups/") {
		bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, "⏳ Link grup terdeteksi, mencoba metode khusus..."))
		filePath, err := ScrapeFacebookGroup(finalURL)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengambil gambar dari grup: "+err.Error()))
			return
		}
		photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FilePath(filePath))
		bot.Send(photo)
		DeleteDirectory(filepath.Dir(filePath))
	} else {
		bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, "⏳ Link ditemukan, sedang mengambil gambar..."))
		filePaths, err := DownloadImagesFromURL(finalURL)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Gagal mengambil gambar: "+err.Error()))
			return
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
		if len(filePaths) > 0 {
			DeleteDirectory(filepath.Dir(filePaths[0]))
		}
	}
}

func handleStatusCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	hostInfo, _ := host.Info()
	cpuCounts, _ := cpu.Counts(true)
	cpuUsage, _ := cpu.Percent(time.Second, false)
	ramInfo, _ := mem.VirtualMemory()
	diskInfo, _ := disk.Usage("/")
	netIO, _ := net.IOCounters(false)

	var totalTraffic uint64
	var bytesSent uint64
	var bytesRecv uint64
	if len(netIO) > 0 {
		bytesSent = netIO[0].BytesSent
		bytesRecv = netIO[0].BytesRecv
		totalTraffic = bytesSent + bytesRecv
	}

	pid := int32(os.Getpid())
	proc, _ := process.NewProcess(pid)
	procRAMInfo, _ := proc.MemoryInfo()

	statusText := fmt.Sprintf(`
			⚙️ *System:*
			├─ CPU: %.2f%% (%d-core)
			├─ RAM: %s / %s (%.2f%%)
			├─ Disk: %s / %s (%.2f%%)
			└─ Uptime: %s
			
			🐹 *App:*
			├─ Latency: 0 ms
			├─ Active Workers: 0
			└─ RAM Usage: %s
			
			🌐 *Networks:*
			├─ In: %s
			├─ Out: %s
			└─ Total Traffic: %s`,
		cpuUsage[0],
		cpuCounts,
		formatFileSize(int64(ramInfo.Used)),
		formatFileSize(int64(ramInfo.Total)),
		ramInfo.UsedPercent,
		formatFileSize(int64(diskInfo.Used)),
		formatFileSize(int64(diskInfo.Total)),
		diskInfo.UsedPercent,
		formatUptime(hostInfo.Uptime),
		formatFileSize(int64(procRAMInfo.RSS)),
		formatFileSize(int64(bytesRecv)),
		formatFileSize(int64(bytesSent)),
		formatFileSize(int64(totalTraffic)),
	)

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, statusText)
	msgConfig.ParseMode = "Markdown"
	bot.Send(msgConfig)
}
