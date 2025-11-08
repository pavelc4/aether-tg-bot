package handlers

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/internal/downloader"
)

func handleDownloadGeneric(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	handleDownloadVideo(bot, msg)
}

func handleDownloadAudio(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		sendText(bot, msg.Chat.ID, "❌ Usage: /mp [URL]")
		return
	}

	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Downloading audio..."))

	start := time.Now()

	filePaths, size, provider, err := downloader.DownloadAudioWithProgress(
		args,
		bot,
		msg.Chat.ID,
		processingMsg.MessageID,
		msg.From.ID,
	)
	if err != nil {
		errorText := fmt.Sprintf("❌ Download failed: %v", err)
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, errorText)
		bot.Send(edit)
		return
	}

	log.Printf(" Downloaded audio via %s: %d files, %.2f MB", provider, len(filePaths), float64(size)/(1024*1024))

	defer downloader.CleanupTempFiles(filePaths)

	source := DetectSource(args)
	duration := time.Since(start)
	username := msg.From.UserName
	if username == "" {
		username = msg.From.FirstName
	}

	caption := BuildMediaCaption(source, args, "Audio", size, duration, username)

	sendMediaGroup(bot, msg.Chat.ID, filePaths, caption, false)

	deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)
}

func handleDownloadVideo(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		sendText(bot, msg.Chat.ID, "❌ Usage: /video [URL]")
		return
	}

	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Downloading video..."))

	start := time.Now()

	filePaths, size, provider, err := downloader.DownloadVideoWithProgress(
		args,
		bot,
		msg.Chat.ID,
		processingMsg.MessageID,
		msg.From.ID,
	)
	if err != nil {
		errorText := fmt.Sprintf("❌ Download failed: %v", err)
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, errorText)
		bot.Send(edit)
		return
	}

	log.Printf(" Downloaded video via %s: %d files, %.2f MB", provider, len(filePaths), float64(size)/(1024*1024))

	defer downloader.CleanupTempFiles(filePaths)

	source := DetectSource(args)
	duration := time.Since(start)
	username := msg.From.UserName
	if username == "" {
		username = msg.From.FirstName
	}

	caption := BuildMediaCaption(source, args, "Video", size, duration, username)

	sendMediaGroup(bot, msg.Chat.ID, filePaths, caption, true)

	deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)
}
