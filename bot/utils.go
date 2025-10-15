package bot

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
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

// ==================== Constants ====================

const (
	speedTestURL      = "https://speed.cloudflare.com/__down?bytes=10000000"
	speedTestTimeout  = 30 * time.Second
	latencyTestURL    = "https://www.google.com"
	progressBarFilled = "█"
	progressBarEmpty  = "░"
	progressBarLength = 20
)

// ==================== Types ====================

type DownloadProgress struct {
	Percentage float64
	Speed      string
	ETA        string
	Downloaded string
	TotalSize  string
	Status     string
}

type SystemInfo struct {
	HostInfo    *host.InfoStat
	CPUCounts   int
	CPUPhysical int
	CPUUsage    float64
	RAMInfo     *mem.VirtualMemoryStat
	DiskInfo    *disk.UsageStat
	BytesSent   uint64
	BytesRecv   uint64
	ProcRAM     int64
	ProcCPU     float64
	BotUptime   time.Duration
	GoRoutines  int
	HeapAlloc   uint64
	NumGC       uint32
}

type SpeedTestResult struct {
	DownloadSpeed float64
	Latency       time.Duration
	Error         error
}

// ==================== Variables ====================

var (
	botStartTime = time.Now()
	ownerID      int64
	ownerIDOnce  sync.Once

	markdownV2Replacer = strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
		"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
		">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
		".", "\\.", "!", "\\!",
	)

	supportedPlatforms = []string{
		"Bilibili", "Bluesky", "Dailymotion", "Facebook",
		"Instagram", "Loom", "OK", "Pinterest", "Newgrounds",
		"Reddit", "Rutube", "Snapchat", "Soundcloud",
		"Streamable", "TikTok", "Tumblr", "Twitch",
		"Twitter", "Vimeo", "VK", "Xiaohongshu", "YouTube",
	}
)

// ==================== Authorization ====================

func IsOwner(userID int64) bool {
	ownerIDOnce.Do(func() {
		ownerID = config.GetOwnerID()
	})
	return ownerID != 0 && userID == ownerID
}

// ==================== URL Utilities ====================

func ResolveFinalURL(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve URL failed: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	if finalURL != url {
		log.Printf("URL resolved: %s -> %s", url, finalURL)
	}

	return finalURL, nil
}

// ==================== Formatting Utilities ====================

func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
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

func formatDuration(d time.Duration) string {
	return formatDurationFromSeconds(uint64(d.Seconds()))
}

func formatUptime(uptimeSec uint64) string {
	return formatDurationFromSeconds(uptimeSec)
}

func formatDurationFromSeconds(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, secs)
	case hours > 0:
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	case minutes > 0:
		return fmt.Sprintf("%dm %ds", minutes, secs)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

func EscapeMarkdownV2(s string) string {
	return markdownV2Replacer.Replace(s)
}

func FormatProgressBar(progress float64) string {
	if progress > 100 {
		progress = 100
	}
	if progress < 0 {
		progress = 0
	}

	filled := int(progress / 100 * progressBarLength)
	empty := progressBarLength - filled

	return fmt.Sprintf(
		"%s%s %.1f%%",
		strings.Repeat(progressBarFilled, filled),
		strings.Repeat(progressBarEmpty, empty),
		progress,
	)
}

// ==================== User Utilities ====================

func GetUserName(msg *tgbotapi.Message) string {
	if msg.From.UserName != "" {
		return "@" + msg.From.UserName
	}
	return msg.From.FirstName
}

// ==================== Progress Bar ====================

func BuildProgressMessage(source string, prog DownloadProgress) string {
	progressBar := FormatProgressBar(prog.Percentage)

	msg := fmt.Sprintf(
		"⏬ *Downloading from %s*\n\n"+
			"`%s`\n\n"+
			"📊 *Progress:* `%s`\n"+
			"⚡ *Speed:* `%s`\n"+
			"⏱️ *ETA:* `%s`",
		EscapeMarkdownV2(source),
		progressBar,
		EscapeMarkdownV2(prog.Downloaded),
		EscapeMarkdownV2(prog.Speed),
		EscapeMarkdownV2(prog.ETA),
	)

	if prog.TotalSize != "" {
		msg += fmt.Sprintf("\n💾 *Total:* `%s`", EscapeMarkdownV2(prog.TotalSize))
	}

	return msg
}

func UpdateProgressMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, source string, prog DownloadProgress) {
	text := BuildProgressMessage(source, prog)

	editConfig := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editConfig.ParseMode = "MarkdownV2"

	if _, err := bot.Request(editConfig); err != nil {
		if !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("Failed to update progress: %v", err)
		}
	}
}

// ==================== Speed Test ====================

func RunSpeedTest() SpeedTestResult {
	ctx, cancel := context.WithTimeout(context.Background(), speedTestTimeout)
	defer cancel()

	result := SpeedTestResult{}

	if latency, err := testLatency(ctx); err == nil {
		result.Latency = latency
	}

	downloadSpeed, err := testDownloadSpeed(ctx)
	if err != nil {
		result.Error = err
		return result
	}

	result.DownloadSpeed = downloadSpeed
	return result
}

func testLatency(ctx context.Context) (time.Duration, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "HEAD", latencyTestURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return time.Since(start), nil
}

func testDownloadSpeed(ctx context.Context) (float64, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", speedTestURL, nil)
	if err != nil {
		return 0, fmt.Errorf("speed test request failed: %w", err)
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("speed test request failed: %w", err)
	}
	defer resp.Body.Close()

	downloaded, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("speed test read failed: %w", err)
	}

	duration := time.Since(start).Seconds()
	if duration <= 0 {
		return 0, fmt.Errorf("invalid test duration")
	}

	return float64(downloaded) / duration / 1024 / 1024, nil
}

// ==================== System Information ====================

func gatherSystemInfo() SystemInfo {
	info := SystemInfo{}

	if hostInfo, err := host.Info(); err == nil {
		info.HostInfo = hostInfo
	} else {
		log.Printf("Error getting host info: %v", err)
		info.HostInfo = &host.InfoStat{Hostname: "Unknown", OS: "Unknown"}
	}

	info.CPUCounts, _ = cpu.Counts(true)
	info.CPUPhysical, _ = cpu.Counts(false)

	if cpuUsage, err := cpu.Percent(time.Second, false); err == nil && len(cpuUsage) > 0 {
		info.CPUUsage = cpuUsage[0]
	}

	if ramInfo, err := mem.VirtualMemory(); err == nil {
		info.RAMInfo = ramInfo
	} else {
		log.Printf("Error getting RAM info: %v", err)
		info.RAMInfo = &mem.VirtualMemoryStat{}
	}

	if diskInfo, err := disk.Usage("/"); err == nil {
		info.DiskInfo = diskInfo
	} else {
		log.Printf("Error getting disk info: %v", err)
		info.DiskInfo = &disk.UsageStat{}
	}

	if netIO, err := net.IOCounters(false); err == nil && len(netIO) > 0 {
		info.BytesSent = netIO[0].BytesSent
		info.BytesRecv = netIO[0].BytesRecv
	}

	if proc, err := process.NewProcess(int32(os.Getpid())); err == nil {
		if procRAMInfo, err := proc.MemoryInfo(); err == nil {
			info.ProcRAM = int64(procRAMInfo.RSS)
		}
		info.ProcCPU, _ = proc.CPUPercent()
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	info.GoRoutines = runtime.NumGoroutine()
	info.HeapAlloc = m.HeapAlloc
	info.NumGC = m.NumGC
	info.BotUptime = time.Since(botStartTime)

	return info
}

// ==================== Command Handlers ====================

func HandleStatusCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if !IsOwner(msg.From.ID) {
		sendMessage(bot, msg.Chat.ID, "❌ This command is restricted to bot owner.")
		return
	}

	processingMsg := sendMessage(bot, msg.Chat.ID, "⏳ Gathering system information...")
	if processingMsg == nil {
		return
	}
	defer deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)

	sysInfo := gatherSystemInfo()

	updateProcessingMessage(bot, msg.Chat.ID, processingMsg.MessageID, "⏳ Running speed test...")
	speedTest := RunSpeedTest()

	statusText := buildStatusText(sysInfo, speedTest)
	sendMarkdownMessage(bot, msg.Chat.ID, statusText)
}

func HandleHelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	helpText := "Welcome to Aether Bot ✨\n\n" +
		"This bot makes it easy to download content from various social media platforms.\n\n" +
		"Just send a link from a supported platform, and the bot will download it for you.\n\n" +
		"*Available Commands:*\n" +
		" • `/dl [URL]` - Download content (alias: `/mp`)\n" +
		" • `/video [URL]` - Download video only\n" +
		" • `/tikaudio [URL]` - Download TikTok audio\n" +
		" • `/help` - Show this help message\n" +
		" • `/support` - List supported platforms\n" +
		" • `/stats` - System status (owner only)\n\n" +
		"Fun fact: This bot is written entirely in Go 🐹"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Developer", "https://t.me/Pavellc"),
			tgbotapi.NewInlineKeyboardButtonURL("Donate", "https://t.me/pavellc"),
		),
	)

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, helpText)
	msgConfig.ReplyMarkup = keyboard
	bot.Send(msgConfig)
}

func HandleSupportCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	supportText := "*Supported Platforms:*\n• " + strings.Join(supportedPlatforms, "\n• ")
	sendMarkdownMessage(bot, msg.Chat.ID, supportText)
}

// ==================== Message Builders ====================

func buildStatusText(info SystemInfo, speedTest SpeedTestResult) string {
	speedTestInfo := "N/A"
	if speedTest.Error == nil {
		speedTestInfo = fmt.Sprintf(
			"├─ Download: `%.2f MB/s`\n└─ Latency: `%dms`",
			speedTest.DownloadSpeed,
			speedTest.Latency.Milliseconds(),
		)
	} else {
		speedTestInfo = fmt.Sprintf("└─ Error: `%s`", speedTest.Error.Error())
	}

	return fmt.Sprintf(
		"🖥️ *System Information*\n"+
			"├─ OS: `%s`\n"+
			"├─ Hostname: `%s`\n"+
			"└─ Uptime: `%s`\n\n"+
			"⚙️ *CPU*\n"+
			"├─ Cores: `%d` (`%d` threads)\n"+
			"└─ Usage: `%.2f%%`\n\n"+
			"💾 *Memory*\n"+
			"├─ Used: `%s / %s` (`%.1f%%`)\n"+
			"└─ Available: `%s`\n\n"+
			"💿 *Disk (/)*\n"+
			"├─ Used: `%s / %s` (`%.1f%%`)\n"+
			"└─ Free: `%s`\n\n"+
			"🌐 *Network*\n"+
			"├─ Sent: `%s`\n"+
			"└─ Received: `%s`\n\n"+
			"🚀 *Speed Test*\n"+
			"%s\n\n"+
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
		info.HostInfo.OS,
		info.HostInfo.Hostname,
		formatUptime(info.HostInfo.Uptime),
		info.CPUPhysical,
		info.CPUCounts,
		info.CPUUsage,
		FormatFileSize(int64(info.RAMInfo.Used)),
		FormatFileSize(int64(info.RAMInfo.Total)),
		info.RAMInfo.UsedPercent,
		FormatFileSize(int64(info.RAMInfo.Available)),
		FormatFileSize(int64(info.DiskInfo.Used)),
		FormatFileSize(int64(info.DiskInfo.Total)),
		info.DiskInfo.UsedPercent,
		FormatFileSize(int64(info.DiskInfo.Free)),
		FormatFileSize(int64(info.BytesSent)),
		FormatFileSize(int64(info.BytesRecv)),
		speedTestInfo,
		formatDuration(info.BotUptime),
		os.Getpid(),
		info.ProcCPU,
		FormatFileSize(info.ProcRAM),
		runtime.Version(),
		info.GoRoutines,
		FormatFileSize(int64(info.HeapAlloc)),
		info.NumGC,
	)
}

func BuildMediaCaption(source, url, fileType string, fileSize int64, duration time.Duration, user string) string {
	return fmt.Sprintf(
		`✅ *%s Downloaded Successfully*`+"\n\n"+
			"🔗 *Source:* [%s](%s)"+"\n"+
			"💾 *Size:* %s"+"\n"+
			"⏱️ *Processing Time:* %s"+"\n"+
			"👤 *By:* %s",
		EscapeMarkdownV2(fileType),
		EscapeMarkdownV2(strings.ToLower(source)),
		EscapeMarkdownV2(url),
		EscapeMarkdownV2(FormatFileSize(fileSize)),
		EscapeMarkdownV2(duration.String()),
		EscapeMarkdownV2(user),
	)
}

// DeleteDirectory removes directory and logs errors
func DeleteDirectory(path string) {
	if err := os.RemoveAll(path); err != nil {
		log.Printf("Warning: Failed to delete directory %s: %v", path, err)
	}
}

// ==================== Telegram Helper Functions ====================

func deleteMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	if _, err := bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID)); err != nil {
		log.Printf("Failed to delete message: %v", err)
	}
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) *tgbotapi.Message {
	msg, err := bot.Send(tgbotapi.NewMessage(chatID, text))
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		return nil
	}
	return &msg
}

func sendMarkdownMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msgConfig := tgbotapi.NewMessage(chatID, text)
	msgConfig.ParseMode = "MarkdownV2"

	if _, err := bot.Send(msgConfig); err != nil {
		log.Printf("Failed to send markdown message: %v", err)
		msgConfig.ParseMode = ""
		msgConfig.Text = strings.ReplaceAll(strings.ReplaceAll(text, "`", ""), "*", "")
		bot.Send(msgConfig)
	}
}

func updateProcessingMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, text string) {
	editConfig := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if _, err := bot.Request(editConfig); err != nil {
		log.Printf("Failed to update message: %v", err)
	}
}
