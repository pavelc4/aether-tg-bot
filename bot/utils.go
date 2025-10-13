package bot

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

var botStartTime = time.Now()

func IsOwner(userID int64) bool {
	ownerID := config.GetOwnerID()
	if ownerID == 0 {
		return false
	}
	return userID == ownerID
}

func ResolveFinalURL(url string) (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("gagal membuka link: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	fmt.Printf("URL asli: %s -> URL final: %s\n", url, finalURL)

	return finalURL, nil
}

func FormatFileSize(size int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)
	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/TB)
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

func GetUserName(msg *tgbotapi.Message) string {
	if msg.From.UserName != "" {
		return "@" + msg.From.UserName
	}
	return msg.From.FirstName
}

func DeleteDirectory(path string) {
	_ = os.RemoveAll(path)
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

func formatDuration(d time.Duration) string {
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

var markdownV2Replacer = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
	"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
	"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
)

func EscapeMarkdownV2(s string) string {
	return markdownV2Replacer.Replace(s)
}

func HandleStatusCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Check if user is owner
	if !IsOwner(msg.From.ID) {
		msgConfig := tgbotapi.NewMessage(msg.Chat.ID, "❌ Perintah ini hanya dapat digunakan oleh owner bot.")
		bot.Send(msgConfig)
		return
	}

	// Send processing message
	processingMsg := tgbotapi.NewMessage(msg.Chat.ID, "⏳ Mengambil informasi sistem...")
	sentMsg, _ := bot.Send(processingMsg)

	// Defer cleanup and panic recovery
	defer func() {
		if r := recover(); r != nil {
			errorMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Error: %v", r))
			bot.Send(errorMsg)
			log.Printf("Panic in HandleStatusCommand: %v\n", r)
		}
	}()

	// Gather system info with error handling
	hostInfo, err := host.Info()
	if err != nil {
		log.Printf("Error getting host info: %v", err)
		hostInfo = &host.InfoStat{Hostname: "Unknown", OS: "Unknown", Uptime: 0}
	}

	cpuCounts, _ := cpu.Counts(true)
	cpuUsage, err := cpu.Percent(time.Second, false)
	if err != nil || len(cpuUsage) == 0 {
		cpuUsage = []float64{0.0}
	}

	ramInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting RAM info: %v", err)
		ramInfo = &mem.VirtualMemoryStat{}
	}

	diskInfo, err := disk.Usage("/")
	if err != nil {
		log.Printf("Error getting disk info: %v", err)
		diskInfo = &disk.UsageStat{}
	}

	netIO, _ := net.IOCounters(false)
	var bytesSent, bytesRecv uint64
	if len(netIO) > 0 {
		bytesSent = netIO[0].BytesSent
		bytesRecv = netIO[0].BytesRecv
	}

	// Process Info
	proc, err := process.NewProcess(int32(os.Getpid()))
	var procRAMUsage int64
	var procCPU float64
	if err == nil {
		if procRAMInfo, err := proc.MemoryInfo(); err == nil {
			procRAMUsage = int64(procRAMInfo.RSS)
		}
		procCPU, _ = proc.CPUPercent()
	}

	// Go Runtime Info
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Bot uptime
	botUptime := time.Since(botStartTime)

	// Build status text
	statusText := fmt.Sprintf(
		"🖥️ *System Information*\n"+
			"├─ OS: `%s`\n"+
			"├─ Hostname: `%s`\n"+
			"└─ Uptime: `%s`\n\n"+

			"⚙️ *CPU*\n"+
			"├─ Cores: `%d`\n"+
			"└─ Usage: `%.2f%%`\n\n"+

			"💾 *Memory*\n"+
			"├─ RAM: `%s / %s` `(%.1f%%)`\n"+
			"└─ Available: `%s`\n\n"+

			"💿 *Disk (/)*\n"+
			"├─ Used: `%s / %s` `(%.1f%%)`\n"+
			"└─ Free: `%s`\n\n"+

			"🌐 *Network*\n"+
			"├─ Sent: `%s`\n"+
			"└─ Received: `%s`\n\n"+

			"🐹 *Bot Process*\n"+
			"├─ Uptime: `%s`\n"+
			"├─ PID: `%d`\n"+
			"├─ CPU: `%.2f%%`\n"+
			"├─ Memory: `%s`\n"+
			"└─ Go Version: `%s`\n\n"+

			"🔧 *Go Runtime*\n"+
			"├─ Goroutines: `%d`\n"+
			"├─ Heap Alloc: `%s`\n"+
			"└─ GC Runs: `%d`",

		// System
		hostInfo.OS,
		hostInfo.Hostname,
		formatUptime(hostInfo.Uptime),

		// CPU
		cpuCounts,
		cpuUsage[0],

		// Memory
		FormatFileSize(int64(ramInfo.Used)),
		FormatFileSize(int64(ramInfo.Total)),
		ramInfo.UsedPercent,
		FormatFileSize(int64(ramInfo.Available)),

		// Disk
		FormatFileSize(int64(diskInfo.Used)),
		FormatFileSize(int64(diskInfo.Total)),
		diskInfo.UsedPercent,
		FormatFileSize(int64(diskInfo.Free)),

		// Network
		FormatFileSize(int64(bytesSent)),
		FormatFileSize(int64(bytesRecv)),

		// Bot Process
		formatDuration(botUptime),
		os.Getpid(),
		procCPU,
		FormatFileSize(procRAMUsage),
		runtime.Version(),

		// Go Runtime
		runtime.NumGoroutine(),
		FormatFileSize(int64(m.HeapAlloc)),
		m.NumGC,
	)

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, statusText)
	msgConfig.ParseMode = "MarkdownV2"

	// Delete processing message
	deleteMsg := tgbotapi.NewDeleteMessage(msg.Chat.ID, sentMsg.MessageID)
	bot.Request(deleteMsg)

	// Send final status
	if _, err := bot.Send(msgConfig); err != nil {
		log.Printf("Error sending status: %v", err)
		// Try sending without markdown if it fails
		msgConfig.ParseMode = ""
		msgConfig.Text = strings.ReplaceAll(statusText, "`", "")
		msgConfig.Text = strings.ReplaceAll(msgConfig.Text, "*", "")
		bot.Send(msgConfig)
	}
}

func BuildMediaCaption(source, url, fileType string, fileSize int64, duration time.Duration, user string) string {
	escapedSource := EscapeMarkdownV2(strings.ToLower(source))
	escapedURL := EscapeMarkdownV2(url)
	escapedFileType := EscapeMarkdownV2(fileType)
	escapedSize := EscapeMarkdownV2(FormatFileSize(fileSize))
	escapedDuration := EscapeMarkdownV2(duration.String())
	escapedUser := EscapeMarkdownV2(user)

	captionFormat := `✅ *%s Berhasil Diunduh*` + "\n\n" +
		"🔗 *Sumber:* [%s](%s)" + "\n" +
		"💾 *Ukuran:* %s" + "\n" +
		"⏱️ *Durasi Proses:* %s" + "\n" +
		"👤 *Oleh:* %s"

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

func HandleHelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	helpText := "Selamat datang di Aether Bot ✨\n\n" +
		"Bot ini Diciptakan Untuk mempermudah Anda dalam mengunduh konten dari berbagai platform sosial media.\n\n" +
		"Cukup kirimkan tautan dari platform yang didukung, dan Bot akan mengunduh kontennya Untuk Anda .\n\n" +
		"Fun fact: Bot ini sepenuhnya ditulis dalam bahasa Go 🐹 \n\n" +
		"Gunakan perintah /support untuk melihat daftar platform yang didukung.\n\n" +
		"Perintah yang tersedia:\n" +
		" • `/help` - Menampilkan pesan bantuan\n" +
		" • `/stats` - Menampilkan status bot (Owner only)\n" +
		" • `/support` - Menampilkan daftar platform yang dapat diunduh.\n" +
		" • `/tikaudio` - Mengunduh audio dari tautan TikTok."

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Developer", "https://t.me/Pavellc"),
			tgbotapi.NewInlineKeyboardButtonURL("Donasi", "https://t.me/pavellc"),
		),
	)

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, helpText)
	msgConfig.ReplyMarkup = inlineKeyboard
	bot.Send(msgConfig)
}

func HandleSupportCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	supportText := "Platform yang didukung:\n" +
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
		"- YouTube"

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, supportText)
	bot.Send(msgConfig)
}
