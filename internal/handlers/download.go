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
	handleDownload(bot, msg, false)
}

func handleDownloadAudio(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	handleDownload(bot, msg, true)
}

func handleDownloadVideo(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	handleDownload(bot, msg, false)
}

func handleDownload(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, audioOnly bool) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		cmd := "/video"
		if audioOnly {
			cmd = "/mp"
		}
		sendText(bot, msg.Chat.ID, fmt.Sprintf("❌ Usage: %s [URL]", cmd))
		return
	}

	mediaType := "video"
	if audioOnly {
		mediaType = "audio"
	}

	processingMsg, _ := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("⏳ Downloading %s...", mediaType)))

	start := time.Now()
	var filePaths []string
	var size int64
	var provider string
	var err error

	if audioOnly {
		filePaths, size, provider, err = downloader.DownloadAudioWithProgress(
			args, bot, msg.Chat.ID, processingMsg.MessageID, msg.From.ID,
		)
	} else {
		filePaths, size, provider, err = downloader.DownloadVideoWithProgress(
			args, bot, msg.Chat.ID, processingMsg.MessageID, msg.From.ID,
		)
	}

	if err != nil {
		errorText := fmt.Sprintf("❌ Download failed: %v", err)
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, processingMsg.MessageID, errorText)
		bot.Send(edit)
		return
	}

	log.Printf(" Downloaded %s via %s: %d files, %.2f MB", mediaType, provider, len(filePaths), float64(size)/(1024*1024))

	defer downloader.CleanupTempFiles(filePaths)

	source := DetectSource(args)
	duration := time.Since(start)
	username := msg.From.UserName
	if username == "" {
		username = msg.From.FirstName
	}

	captionType := "Video"
	if audioOnly {
		captionType = "Audio"
	}
	caption := BuildMediaCaption(source, args, captionType, size, duration, username)

	sendMediaGroupWithProgress(bot, msg.Chat.ID, filePaths, caption, !audioOnly, processingMsg.MessageID, username)
}
