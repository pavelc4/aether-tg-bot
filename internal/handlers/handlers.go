package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/internal/downloader"
	"github.com/pavelc4/aether-tg-bot/internal/stats"
	"github.com/pavelc4/aether-tg-bot/pkg/utils"
)

var (
	urlRegex = regexp.MustCompile(`(https?://[^\s]+)`)
)

func HandleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		handleStart(bot, msg)
	case "help":
		handleHelp(bot, msg)
	case "support":
		handleSupport(bot, msg)
	case "speedtest":
		handleSpeedTest(bot, msg)
	case "stats":
		handleStats(bot, msg)
	case "mp":
		handleDownloadAudio(bot, msg)
	case "video":
		handleDownloadVideo(bot, msg)
	case "dl":
		handleDownloadGeneric(bot, msg)
	default:
		sendText(bot, msg.Chat.ID, "❌ Unknown command. Type /help to see available commands.")
	}
}

func HandleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	url := urlRegex.FindString(msg.Text)
	if url == "" {
		return
	}

	processingMsg, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Processing link, please wait..."))
	if err != nil {
		log.Printf("Failed to send processing message: %v", err)
		return
	}

	defer deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)

	start := time.Now()

	filePaths, size, provider, err := downloader.DownloadVideo(url)
	if err != nil {
		sendText(bot, msg.Chat.ID, fmt.Sprintf("❌ Download failed: %v", err))
		return
	}

	log.Printf("✅ Downloaded via %s: %d files, %.2f MB", provider, len(filePaths), float64(size)/(1024*1024))

	defer downloader.CleanupTempFiles(filePaths)

	source := DetectSource(url)
	duration := time.Since(start)
	username := msg.From.UserName
	if username == "" {
		username = msg.From.FirstName
	}

	caption := BuildMediaCaption(source, url, "Video", size, duration, username)

	sendMediaGroup(bot, msg.Chat.ID, filePaths, caption, true)
}

func handleStart(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := `👋 *Welcome to Aether Downloader Bot!*

I can help you download media from various platforms.

📹 *Supported platforms:*
• YouTube
• TikTok
• Instagram
• Twitter/X
• And more!

🚀 *How to use:*
1. Send me a URL to download video
2. Use /mp [URL] to download audio only
3. Use /help for more commands


Send me a link to get started!`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	if _, err := bot.Send(reply); err != nil {
		log.Printf("Failed to send start message: %v", err)
	}
}

func handleHelp(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := `📚 *Available Commands:*

/start - Start the bot
/help - Show this help message
/mp [URL] - Download audio only
/video [URL] - Download video
/dl [URL] - Generic download
/stats - Show bot statistics
/support - Get support
/speedtest - Test Internet speed (owner Only)
/stats - Show bot statistics (owner only)

💡 *Quick Tips:*
• Just send a URL to download video
• Bot uses Cobalt API first, then falls back to yt-dlp
• Adaptive aria2c enabled for faster downloads

Need help? Contact @pavelc`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	if _, err := bot.Send(reply); err != nil {
		log.Printf("Failed to send help message: %v", err)
		sendText(bot, msg.Chat.ID, strings.ReplaceAll(text, "*", ""))
	}
}

func handleSupport(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	sendText(bot, msg.Chat.ID, "💬 For support, contact: @your_username")
}

func handleDownloadAudio(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		sendText(bot, msg.Chat.ID, "❌ Usage: /mp [URL]")
		return
	}

	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Downloading audio..."))
	defer deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)

	start := time.Now()

	filePaths, size, provider, err := downloader.DownloadAudio(args)
	if err != nil {
		sendText(bot, msg.Chat.ID, fmt.Sprintf("❌ Download failed: %v", err))
		return
	}

	log.Printf("✅ Downloaded audio via %s: %d files, %.2f MB", provider, len(filePaths), float64(size)/(1024*1024))

	defer downloader.CleanupTempFiles(filePaths)

	source := DetectSource(args)
	duration := time.Since(start)
	username := msg.From.UserName
	if username == "" {
		username = msg.From.FirstName
	}

	caption := BuildMediaCaption(source, args, "Audio", size, duration, username)

	sendMediaGroup(bot, msg.Chat.ID, filePaths, caption, false)
}

func handleDownloadVideo(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		sendText(bot, msg.Chat.ID, "❌ Usage: /video [URL]")
		return
	}

	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Downloading video..."))
	defer deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)

	start := time.Now()

	filePaths, size, provider, err := downloader.DownloadVideo(args)
	if err != nil {
		sendText(bot, msg.Chat.ID, fmt.Sprintf("❌ Download failed: %v", err))
		return
	}

	log.Printf("✅ Downloaded video via %s: %d files, %.2f MB", provider, len(filePaths), float64(size)/(1024*1024))

	defer downloader.CleanupTempFiles(filePaths)

	source := DetectSource(args)
	duration := time.Since(start)
	username := msg.From.UserName
	if username == "" {
		username = msg.From.FirstName
	}

	caption := BuildMediaCaption(source, args, "Video", size, duration, username)

	sendMediaGroup(bot, msg.Chat.ID, filePaths, caption, true)
}

func handleDownloadGeneric(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	handleDownloadVideo(bot, msg)
}

func handleSpeedTest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if !isOwner(msg.From.ID) {
		sendText(bot, msg.Chat.ID, "❌ This command is only available to the bot owner.")
		log.Printf("⚠️  Unauthorized speedtest attempt by user %d (%s)", msg.From.ID, msg.From.UserName)
		return
	}

	statusMsg, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🚀 Running Speed Test"))
	if err != nil {
		log.Printf("Failed to send speedtest status: %v", err)
		return
	}

	start := time.Now()
	result := utils.RunSpeedTest()
	totalDuration := time.Since(start)

	var resultText string
	if result.Error != nil {
		resultText = fmt.Sprintf(
			"❌ *Speed Test Failed*\n\n"+
				"└─ Error: `%v`",
			result.Error,
		)
	} else {
		speedMBps := result.DownloadSpeed / 8
		resultText = fmt.Sprintf(
			"🚀 *Network Speed Test*\n"+
				"├─ *Download:* `%.2f MB/s` (%.2f Mbps)\n"+
				"├─ *Latency:* `%d ms`\n"+
				"├─ *Data Used:* `%s`\n"+
				"├─ *Test Duration:* `%.1fs`\n"+
				"└─ *Total Time:* `%.1fs`\n\n"+
				"_Test server: Cloudflare_",
			speedMBps,
			result.DownloadSpeed,
			result.Latency.Milliseconds(),
			FormatFileSize(result.BytesDownloaded),
			result.Duration.Seconds(),
			totalDuration.Seconds(),
		)
	}

	// Update message with results
	edit := tgbotapi.NewEditMessageText(msg.Chat.ID, statusMsg.MessageID, resultText)
	edit.ParseMode = "Markdown"
	if _, err := bot.Send(edit); err != nil {
		log.Printf("Failed to update speedtest message: %v", err)
		// Fallback: send new message
		reply := tgbotapi.NewMessage(msg.Chat.ID, resultText)
		reply.ParseMode = "Markdown"
		bot.Send(reply)
	}

	log.Printf("✅ Speedtest completed by owner %d: %.2f MB/s (%.2f Mbps), %dms latency",
		msg.From.ID, result.DownloadSpeed/8, result.DownloadSpeed, result.Latency.Milliseconds())
}

func handleStats(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if !isOwner(msg.From.ID) {
		sendText(bot, msg.Chat.ID, "❌ This command is only available to the bot owner.")
		log.Printf("⚠️  Unauthorized stats attempt by user %d (%s)", msg.From.ID, msg.From.UserName)
		return
	}

	statusMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Gathering system information..."))

	sysInfo, err := stats.GetSystemInfo()
	if err != nil {
		sendText(bot, msg.Chat.ID, "❌ Failed to get system information")
		log.Printf("Failed to get system info: %v", err)
		return
	}

	todayStats := stats.GetStats().GetPeriodStats("today")
	weekStats := stats.GetStats().GetPeriodStats("week")
	monthStats := stats.GetStats().GetPeriodStats("month")

	statsText := fmt.Sprintf(
		"🖥️ *System Information*\n"+
			"├─ OS: `%s`\n"+
			"├─ Hostname: `%s`\n"+
			"└─ Uptime: `%s`\n\n"+

			"⚙️ *CPU*\n"+
			"├─ Cores: `%d`\n"+
			"└─ Usage: `%.2f%%`\n\n"+

			"💾 *Memory*\n"+
			"├─ Used: `%s / %s` (%.1f%%)\n"+
			"└─ Available: `%s`\n\n"+

			"💿 *Disk (/)*\n"+
			"├─ Used: `%s / %s` (%.1f%%)\n"+
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
			"└─ GC Runs: `%d`\n\n"+

			"📊 *Download Stats*\n"+
			"├─ Today: `%s`\n"+
			"├─ This Week: `%s`\n"+
			"└─ This Month: `%s`",

		sysInfo.OS,
		sysInfo.Hostname,
		formatUptime(sysInfo.SystemUptime),

		sysInfo.CPUCores,
		sysInfo.CPUUsage,

		FormatFileSize(int64(sysInfo.MemUsed)),
		FormatFileSize(int64(sysInfo.MemTotal)),
		sysInfo.MemPercent,
		FormatFileSize(int64(sysInfo.MemAvailable)),

		FormatFileSize(int64(sysInfo.DiskUsed)),
		FormatFileSize(int64(sysInfo.DiskTotal)),
		sysInfo.DiskPercent,
		FormatFileSize(int64(sysInfo.DiskFree)),

		FormatFileSize(int64(sysInfo.NetSent)),
		FormatFileSize(int64(sysInfo.NetRecv)),

		formatUptime(sysInfo.ProcessUptime),
		sysInfo.ProcessPID,
		sysInfo.ProcessCPU,
		FormatFileSize(int64(sysInfo.ProcessMem)),
		sysInfo.GoVersion,

		sysInfo.Goroutines,
		FormatFileSize(int64(sysInfo.HeapAlloc)),
		sysInfo.GCRuns,

		formatPeriodStats(todayStats),
		formatPeriodStats(weekStats),
		formatPeriodStats(monthStats),
	)

	edit := tgbotapi.NewEditMessageText(msg.Chat.ID, statusMsg.MessageID, statsText)
	edit.ParseMode = "Markdown"
	if _, err := bot.Send(edit); err != nil {
		log.Printf("Failed to update stats message: %v", err)
		reply := tgbotapi.NewMessage(msg.Chat.ID, statsText)
		reply.ParseMode = "Markdown"
		bot.Send(reply)
	}

	log.Printf("✅ Stats viewed by owner %d", msg.From.ID)
}

func formatPeriodStats(stats *stats.PeriodStats) string {
	if stats == nil {
		return "No data"
	}
	return fmt.Sprintf("%d downloads (%s)", stats.Downloads, FormatFileSize(stats.Bytes))
}
