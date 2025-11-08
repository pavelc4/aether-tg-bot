package handlers

import (
	"fmt"
	"log"
	"regexp"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/internal/downloader"
)

var (
	urlRegex = regexp.MustCompile(`(https?://[^\s]+)`)
)

var commandHandlers = map[string]func(*tgbotapi.BotAPI, *tgbotapi.Message){
	"start":     handleStart,
	"help":      handleHelp,
	"speedtest": handleSpeedTest,
	"stats":     handleStats,
	"mp":        handleDownloadAudio,
	"video":     handleDownloadVideo,
	"dl":        handleDownloadGeneric,
}

func HandleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	cmd := msg.Command()
	if handler, exists := commandHandlers[cmd]; exists {
		handler(bot, msg)
	} else {
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

	start := time.Now()
	username := msg.From.UserName
	if username == "" {
		username = msg.From.FirstName
	}

	filePaths, size, provider, err := downloader.DownloadVideoWithProgressDetailed(url, bot, msg.Chat.ID, processingMsg.MessageID, username)
	if err != nil {
		errorText := fmt.Sprintf("❌ Download failed: %v", err)
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, errorText)
		bot.Send(edit)
		return
	}

	log.Printf("Downloaded via %s: %d files, %.2f MB", provider, len(filePaths), float64(size)/(1024*1024))
	defer downloader.CleanupTempFiles(filePaths)

	source := DetectSource(url)
	duration := time.Since(start)
	caption := BuildMediaCaption(source, url, "Video", size, duration, username)

	sendMediaGroup(bot, msg.Chat.ID, filePaths, caption, true)
	deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)
}
