package bot

import (
	"fmt"
	"net/http"
	"os"
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

var markdownV2Replacer = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
	"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
	"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
)

func EscapeMarkdownV2(s string) string {
	return markdownV2Replacer.Replace(s)
}

func HandleStatusCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
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

	statusText := fmt.Sprintf("⚙️ *System:*\n"+
		"├─ CPU: `%.2f%%` `(%d-core)`\n"+
		"├─ RAM: `%s / %s` `(%.2f%%)`\n"+
		"├─ Disk: `%s / %s` `(%.2f%%)`\n"+
		"└─ Uptime: `%s`\n\n"+
		"🐹 *App: *\n"+
		"└─ RAM Usage: `%s`\n\n"+
		"🌐 *Networks: *\n"+
		"├─ In: `%s`\n"+
		"├─ Out: `%s`\n"+
		"└─ Total Traffic: `%s`",
		cpuUsage[0], cpuCounts,
		FormatFileSize(int64(ramInfo.Used)), FormatFileSize(int64(ramInfo.Total)), ramInfo.UsedPercent,
		FormatFileSize(int64(diskInfo.Used)), FormatFileSize(int64(diskInfo.Total)), diskInfo.UsedPercent,
		formatUptime(hostInfo.Uptime),
		FormatFileSize(int64(procRAMInfo.RSS)),
		FormatFileSize(int64(bytesRecv)),
		FormatFileSize(int64(bytesSent)),
		FormatFileSize(int64(totalTraffic)),
	)

	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, statusText)
	msgConfig.ParseMode = "MarkdownV2"
	bot.Send(msgConfig)
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
		" • `/stats` - Menampilkan status bot\n" +
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
